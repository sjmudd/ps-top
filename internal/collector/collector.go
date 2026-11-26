// Package collector collects every metric on its own ticker, one ring of
// snapshots per metric, and answers requests for the snapshots a metric type
// needs: the latest for CurrentValue, the start and end of a window for
// DifferenceValues. A lock covers the rings only: queries run without it, so
// a request is answered at once, also while a collection is in progress, from
// the snapshots already held. Snapshots are never changed once in a ring.
package collector

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/sjmudd/ps-top/v2/internal/metric"
	"github.com/sjmudd/ps-top/v2/internal/registry"
	"github.com/sjmudd/ps-top/v2/internal/snapshot"
	"github.com/sjmudd/ps-top/v2/internal/store"
)

// Metric is one metric and how to collect its entries.
type Metric struct {
	Def     metric.Definition
	Collect func(*sql.DB) ([]snapshot.Entry, error)
	// Stats is the metric's entry in the metrics registry: the collector
	// records the snapshot stage and its ring there. Nil gives the metric an
	// entry of its own, not in any registry.
	Stats *registry.Collector
}

// Window is the reply to a request: the metric type asked for and its
// snapshots, oldest first, read only. CurrentValue has the latest snapshot.
// DifferenceValues has the window's start and end. Snapshots is empty until
// enough snapshots are held.
type Window struct {
	Type      metric.MetricType
	Snapshots []*snapshot.Snapshot
}

// Ready reports whether w has the snapshots its type needs.
func (w Window) Ready() bool { return len(w.Snapshots) == w.Type.Snapshots() }

type reply struct {
	w   Window
	err error
}

// Collector holds one ring per metric.
type Collector struct {
	db       *sql.DB
	interval time.Duration
	metrics  []Metric
	defs     map[string]metric.Definition
	// mu covers rings: the write lock to add a snapshot, resize or trim,
	// the read lock to answer a request. The map itself is fixed by New.
	mu    sync.RWMutex
	rings map[string]*store.Ring
	slow  slowCollection // the latest collection longer than interval, under mu

	data     chan struct{} // closed when the first collection is done
	dataOnce sync.Once
	stopped  chan struct{} // closed when Run returns, after runErr is set
	runErr   error
}

// New returns a collector that collects metrics every interval. Each
// metric's ring holds what the largest of its types needs over a window of n
// intervals: 1 snapshot for CurrentValue, n+1 for DifferenceValues.
func New(db *sql.DB, interval time.Duration, n int, metrics []Metric) (*Collector, error) {
	if interval <= 0 {
		return nil, fmt.Errorf("collector: interval %v, need more than 0", interval)
	}
	if n < 1 {
		return nil, fmt.Errorf("collector: window of %d intervals, need at least 1", n)
	}
	if len(metrics) == 0 {
		return nil, errors.New("collector: no metrics")
	}
	c := &Collector{
		db:       db,
		interval: interval,
		metrics:  slices.Clone(metrics),
		rings:    make(map[string]*store.Ring, len(metrics)),
		defs:     make(map[string]metric.Definition, len(metrics)),
		data:     make(chan struct{}),
		stopped:  make(chan struct{}),
	}
	for i := range c.metrics {
		m := &c.metrics[i]
		if err := m.Def.Validate(); err != nil {
			return nil, err
		}
		if m.Stats == nil {
			m.Stats = new(registry.Collector)
		}
		if _, dup := c.rings[m.Def.Name]; dup {
			return nil, fmt.Errorf("collector: duplicate metric %q", m.Def.Name)
		}
		size := ringSize(m.Def, n)
		r, err := store.New(size)
		if err != nil {
			return nil, err
		}
		c.rings[m.Def.Name] = r
		c.defs[m.Def.Name] = m.Def
		m.Stats.RingCapacity.Store(uint64(size))
	}
	return c, nil
}

// Run collects at once, then every interval, until ctx is done (returns nil)
// or a collection fails (returns its error). Call it once.
func (c *Collector) Run(ctx context.Context) (err error) {
	defer func() {
		c.runErr = err
		close(c.stopped)
	}()
	tick := time.NewTicker(c.interval)
	defer tick.Stop()
	if err := c.collect(); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
			if err := c.collect(); err != nil {
				return err
			}
		}
	}
}

// ringSize returns the snapshots a ring of def needs over a window of n
// intervals: what the largest of its types needs.
func ringSize(def metric.Definition, n int) int {
	size := 0
	for _, t := range def.Types {
		size = max(size, t.RingSize(n))
	}
	return size
}

// SetWindow resizes every ring for a window of n intervals, as New sizes
// them: a ring grows keeping its snapshots, and shrinks dropping the oldest.
// Until a grown window fills, Window answers from the snapshots held. It
// returns ctx's error if ctx is already done.
func (c *Collector) SetWindow(ctx context.Context, n int) error {
	if n < 1 {
		return fmt.Errorf("collector: window of %d intervals, need at least 1", n)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, m := range c.metrics {
		size := ringSize(m.Def, n)
		r := c.rings[m.Def.Name]
		if err := r.Resize(size); err != nil {
			return err
		}
		m.Stats.RingCapacity.Store(uint64(size))
		m.Stats.Snapshots.Store(uint64(r.Len()))
	}
	return nil
}

// Reset drops all but the latest snapshot of every ring, so windows start
// again from it: a difference is shown again after the next collection. It
// returns ctx's error if ctx is already done.
func (c *Collector) Reset(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, m := range c.metrics {
		r := c.rings[m.Def.Name]
		r.Trim(1)
		m.Stats.Snapshots.Store(uint64(r.Len()))
	}
	return nil
}

// Window returns the snapshots of metric name that type t needs: for
// CurrentValue the latest, for DifferenceValues the start, n before the
// latest (or the oldest held), and the end, the latest. It does not wait for
// a collection in progress: e.g. with 10 and 20 held while the query for 30
// runs, a window of 3 is 10 and 20. It returns ctx's error if ctx is already
// done.
func (c *Collector) Window(ctx context.Context, name string, t metric.MetricType, n int) (Window, error) {
	if err := ctx.Err(); err != nil {
		return Window{}, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	rep := c.window(name, t, n)
	return rep.w, rep.err
}

// dataWaits is how many collect intervals WaitForData waits at most.
const dataWaits = 3

// WaitForData waits until the first collection is done, so every metric has
// a snapshot, and returns nil, at once if it already is. It waits at most 3
// collect intervals: with a 1s interval, a first collection not done after
// 3s is an error. If Run stops first it returns Run's error, or an error
// saying Run stopped if Run had none. If ctx is done first it returns ctx's
// error. For example, the first --batch output waits for it, later ones do
// not.
func (c *Collector) WaitForData(ctx context.Context) error {
	select {
	case <-c.data:
		return nil
	default:
	}
	limit := dataWaits * c.interval
	timeout := time.NewTimer(limit)
	defer timeout.Stop()
	select {
	case <-timeout.C:
		return fmt.Errorf("collector: no data after %v, %d collect intervals", limit, dataWaits)
	case <-c.data:
		return nil
	case <-c.stopped:
		if c.runErr != nil {
			return c.runErr
		}
		return errors.New("collector: stopped before the first collection")
	case <-ctx.Done():
		return ctx.Err()
	}
}

// slowCollection is a collection that took longer than the collect interval:
// the ticker then drops a tick, so the snapshots are further apart than set.
type slowCollection struct {
	took time.Duration
	at   time.Time // when it ended
}

// SlowCollection returns the latest collection that took longer than the
// collect interval: the time it took and when it ended, or 0 and the zero
// time if none has. For example, 1.2s at a 1s collect interval.
func (c *Collector) SlowCollection() (took time.Duration, at time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.slow.took, c.slow.at
}

// collect takes one snapshot of every metric, in turn, and records it in
// c.slow if it took longer than the collect interval.
func (c *Collector) collect() error {
	start := time.Now()
	defer func() {
		if took := time.Since(start); took > c.interval {
			c.mu.Lock()
			c.slow = slowCollection{took: took, at: time.Now()}
			c.mu.Unlock()
		}
	}()
	for _, m := range c.metrics {
		t := time.Now()
		entries, err := m.Collect(c.db)
		if err != nil {
			return fmt.Errorf("collector: %s: %w", m.Def.Name, err)
		}
		if err := c.store(m, t, entries); err != nil {
			return fmt.Errorf("collector: %s: %w", m.Def.Name, err)
		}
	}
	c.dataOnce.Do(func() { close(c.data) })
	return nil
}

// store builds m's snapshot taken at t and adds it to m's ring, recording the
// snapshot stage and the snapshots held in m.Stats. The snapshot is built
// before the lock is taken: only the add holds it.
func (c *Collector) store(m Metric, t time.Time, entries []snapshot.Entry) error {
	timer := m.Stats.Snapshot.Start()
	defer timer.Stop()
	s, err := snapshot.New(t, len(m.Def.Columns), entries)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	r := c.rings[m.Def.Name]
	if err := r.Add(s); err != nil {
		return err
	}
	m.Stats.Snapshots.Store(uint64(r.Len()))
	return nil
}

// window answers a request for Window. The caller holds c.mu, at least the
// read lock.
func (c *Collector) window(name string, t metric.MetricType, n int) reply {
	r, ok := c.rings[name]
	if !ok {
		return reply{err: fmt.Errorf("collector: no metric %q", name)}
	}
	if !c.defs[name].Accepts(t) {
		return reply{err: fmt.Errorf("collector: metric %q does not accept %v", name, t)}
	}
	w := Window{Type: t}
	held := r.Len()
	switch t {
	case metric.CurrentValue:
		if held >= 1 {
			w.Snapshots = []*snapshot.Snapshot{r.Latest()}
		}
	case metric.DifferenceValues:
		if n < 1 {
			return reply{err: fmt.Errorf("collector: window of %d snapshots, need at least 1", n)}
		}
		if held >= 2 {
			w.Snapshots = []*snapshot.Snapshot{r.At(max(held-1-n, 0)), r.Latest()}
		}
	}
	return reply{w: w}
}
