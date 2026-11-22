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
	"time"

	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/snapshot"
	"github.com/sjmudd/ps-top/internal/store"
)

// Metric is one metric and how to collect its entries.
type Metric struct {
	Def     metric.Definition
	Collect func(*sql.DB) ([]snapshot.Entry, error)
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
		metrics:  metrics,
		rings:    make(map[string]*store.Ring, len(metrics)),
		defs:     make(map[string]metric.Definition, len(metrics)),
		requests: make(chan request),
	}
	for _, m := range metrics {
		if err := m.Def.Validate(); err != nil {
			return nil, err
		}
		if _, dup := c.rings[m.Def.Name]; dup {
			return nil, fmt.Errorf("collector: duplicate metric %q", m.Def.Name)
		}
		size := 0
		for _, t := range m.Def.Types {
			size = max(size, t.RingSize(n))
		}
		r, err := store.New(size)
		if err != nil {
			return nil, err
		}
		c.rings[m.Def.Name] = r
		c.defs[m.Def.Name] = m.Def
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
		}
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
		s, err := snapshot.New(t, len(m.Def.Columns), entries)
		if err != nil {
			return fmt.Errorf("collector: %s: %w", m.Def.Name, err)
		}
		if err := c.rings[m.Def.Name].Add(s); err != nil {
			return fmt.Errorf("collector: %s: %w", m.Def.Name, err)
		}
	}
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
