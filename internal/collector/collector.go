// Package collector collects every metric on its own ticker, one ring of
// snapshots per metric, and answers requests for the snapshots a metric type
// needs: the latest for CurrentValue, the start and end of a window for
// DifferenceValues. The rings are only touched by the goroutine running Run.
package collector

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/registry"
	"github.com/sjmudd/ps-top/internal/snapshot"
	"github.com/sjmudd/ps-top/internal/store"
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

type request struct {
	metric string
	typ    metric.MetricType
	n      int
	reply  chan reply
}

type reply struct {
	w   Window
	err error
}

// Collector holds one ring per metric.
type Collector struct {
	db       *sql.DB
	interval time.Duration
	metrics  []Metric
	rings    map[string]*store.Ring
	defs     map[string]metric.Definition
	requests chan request
	changes  chan change
}

// change is a change to the rings (SetWindow, Reset), made by the goroutine
// running Run.
type change struct {
	apply func() error
	reply chan error
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
		requests: make(chan request),
		changes:  make(chan change),
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

// Run collects at once, then every interval, and answers requests until ctx is
// done (returns nil) or a collection fails (returns its error).
func (c *Collector) Run(ctx context.Context) error {
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
		case req := <-c.requests:
			req.reply <- c.window(req.metric, req.typ, req.n)
		case ch := <-c.changes:
			ch.reply <- ch.apply()
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
// waits until Run makes the change or ctx is done.
func (c *Collector) SetWindow(ctx context.Context, n int) error {
	if n < 1 {
		return fmt.Errorf("collector: window of %d intervals, need at least 1", n)
	}
	return c.change(ctx, func() error {
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
	})
}

// Reset drops all but the latest snapshot of every ring, so windows start
// again from it: a difference is shown again after the next collection. It
// waits until Run makes the change or ctx is done.
func (c *Collector) Reset(ctx context.Context) error {
	return c.change(ctx, func() error {
		for _, m := range c.metrics {
			r := c.rings[m.Def.Name]
			r.Trim(1)
			m.Stats.Snapshots.Store(uint64(r.Len()))
		}
		return nil
	})
}

// change has the goroutine running Run apply a change to the rings, and
// waits for it.
func (c *Collector) change(ctx context.Context, apply func() error) error {
	ch := change{apply: apply, reply: make(chan error, 1)}
	select {
	case c.changes <- ch:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-ch.reply:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Window asks the running collector for the snapshots of metric name that
// type t needs: for CurrentValue the latest, for DifferenceValues the start,
// n before the latest (or the oldest held), and the end, the latest. It waits
// until Run answers or ctx is done.
func (c *Collector) Window(ctx context.Context, name string, t metric.MetricType, n int) (Window, error) {
	req := request{metric: name, typ: t, n: n, reply: make(chan reply, 1)}
	select {
	case c.requests <- req:
	case <-ctx.Done():
		return Window{}, ctx.Err()
	}
	select {
	case rep := <-req.reply:
		return rep.w, rep.err
	case <-ctx.Done():
		return Window{}, ctx.Err()
	}
}

// collect takes one snapshot of every metric, in turn.
func (c *Collector) collect() error {
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
	return nil
}

// store builds m's snapshot taken at t and adds it to m's ring, recording the
// snapshot stage and the snapshots held in m.Stats.
func (c *Collector) store(m Metric, t time.Time, entries []snapshot.Entry) error {
	timer := m.Stats.Snapshot.Start()
	defer timer.Stop()
	s, err := snapshot.New(t, len(m.Def.Columns), entries)
	if err != nil {
		return err
	}
	r := c.rings[m.Def.Name]
	if err := r.Add(s); err != nil {
		return err
	}
	m.Stats.Snapshots.Store(uint64(r.Len()))
	return nil
}

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
