// Package collector collects every metric on its own ticker, one ring of
// snapshots per metric, and answers requests for the start and end snapshots
// of a window. The rings are only touched by the goroutine running Run.
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

// Window is the reply to a request: the start and end snapshots, read only.
// OK is false while fewer than two snapshots are held.
type Window struct {
	Start, End *snapshot.Snapshot
	OK         bool
}

type request struct {
	metric string
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
	requests chan request
}

// New returns a collector that collects metrics every interval into rings of
// ringSize snapshots.
func New(db *sql.DB, interval time.Duration, ringSize int, metrics []Metric) (*Collector, error) {
	if interval <= 0 {
		return nil, fmt.Errorf("collector: interval %v, need more than 0", interval)
	}
	if len(metrics) == 0 {
		return nil, errors.New("collector: no metrics")
	}
	c := &Collector{
		db:       db,
		interval: interval,
		metrics:  metrics,
		rings:    make(map[string]*store.Ring, len(metrics)),
		requests: make(chan request),
	}
	for _, m := range metrics {
		if err := m.Def.Validate(); err != nil {
			return nil, err
		}
		if _, dup := c.rings[m.Def.Name]; dup {
			return nil, fmt.Errorf("collector: duplicate metric %q", m.Def.Name)
		}
		r, err := store.New(ringSize)
		if err != nil {
			return nil, err
		}
		c.rings[m.Def.Name] = r
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
			req.reply <- c.window(req.metric, req.n)
		}
	}
}

// Window asks the running collector for the start snapshot, n before the
// latest (or the oldest held), and the end snapshot, the latest, of metric
// name. It waits until Run answers or ctx is done.
func (c *Collector) Window(ctx context.Context, name string, n int) (Window, error) {
	req := request{metric: name, n: n, reply: make(chan reply, 1)}
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

func (c *Collector) window(name string, n int) reply {
	r, ok := c.rings[name]
	if !ok {
		return reply{err: fmt.Errorf("collector: no metric %q", name)}
	}
	if n < 1 {
		return reply{err: fmt.Errorf("collector: window of %d snapshots, need at least 1", n)}
	}
	held := r.Len()
	if held < 2 {
		return reply{}
	}
	return reply{w: Window{Start: r.At(max(held-1-n, 0)), End: r.Latest(), OK: true}}
}
