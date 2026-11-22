package collector

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
	"unique"

	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/snapshot"
)

var (
	diff    = metric.DifferenceValues
	current = metric.CurrentValue
	def     = metric.Definition{
		Name:    "io",
		Columns: []metric.Column{{Name: "sum_timer_wait", Kind: metric.Counter}},
		Types:   []metric.MetricType{diff},
	}
	gauges = metric.Definition{
		Name:    "memory",
		Columns: []metric.Column{{Name: "current_bytes", Kind: metric.Gauge}},
		Types:   []metric.MetricType{current},
	}
)

// counting returns a metric of d whose single value is 10, 20, 30, ... per
// collection.
func counting(d metric.Definition) Metric {
	var v uint64
	return Metric{Def: d, Collect: func(*sql.DB) ([]snapshot.Entry, error) {
		v += 10
		return []snapshot.Entry{{Key: unique.Make("a"), Values: []uint64{v}}}, nil
	}}
}

func newCollector(t *testing.T, n int, metrics ...Metric) *Collector {
	t.Helper()
	c, err := New(nil, time.Millisecond, n, metrics)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func collect(t *testing.T, c *Collector, times int) {
	t.Helper()
	for range times {
		if err := c.collect(); err != nil {
			t.Fatal(err)
		}
	}
}

func first(s *snapshot.Snapshot) uint64 { return s.Entries[0].Values[0] }

func TestRingSizes(t *testing.T) {
	c := newCollector(t, 9, counting(def), counting(gauges))
	collect(t, c, 15)
	if got := c.rings["io"].Len(); got != 10 {
		t.Errorf("DifferenceValues ring holds %d, want 10 (n+1)", got)
	}
	if got := c.rings["memory"].Len(); got != 1 {
		t.Errorf("CurrentValue ring holds %d, want 1", got)
	}
}

func TestWindowDifference(t *testing.T) {
	c := newCollector(t, 9, counting(def))
	if rep := c.window("io", diff, 5); rep.err != nil || rep.w.Ready() {
		t.Fatalf("no snapshots: %+v, want not ready", rep)
	}
	collect(t, c, 1)
	if rep := c.window("io", diff, 5); rep.err != nil || rep.w.Ready() {
		t.Fatalf("one snapshot: %+v, want not ready", rep)
	}
	collect(t, c, 7)
	// 8 snapshots held: 10 ... 80. n=5: start 30, end 80.
	rep := c.window("io", diff, 5)
	if rep.err != nil || !rep.w.Ready() || rep.w.Type != diff {
		t.Fatalf("n=5: %+v, want ready DifferenceValues", rep)
	}
	if s := rep.w.Snapshots; first(s[0]) != 30 || first(s[1]) != 80 {
		t.Errorf("n=5: start %d end %d, want 30 and 80", first(s[0]), first(s[1]))
	}
	// Fewer held than asked: start is the oldest.
	rep = c.window("io", diff, 20)
	if rep.err != nil || first(rep.w.Snapshots[0]) != 10 {
		t.Errorf("n=20: start %d (%v), want 10 (oldest)", first(rep.w.Snapshots[0]), rep.err)
	}
}

func TestWindowCurrent(t *testing.T) {
	c := newCollector(t, 9, counting(gauges))
	if rep := c.window("memory", current, 9); rep.err != nil || rep.w.Ready() {
		t.Fatalf("no snapshots: %+v, want not ready", rep)
	}
	// One snapshot is enough.
	collect(t, c, 1)
	rep := c.window("memory", current, 9)
	if rep.err != nil || !rep.w.Ready() || first(rep.w.Snapshots[0]) != 10 {
		t.Fatalf("one snapshot: %+v, want ready with 10", rep)
	}
	collect(t, c, 2)
	if rep := c.window("memory", current, 9); first(rep.w.Snapshots[0]) != 30 {
		t.Errorf("latest = %d, want 30", first(rep.w.Snapshots[0]))
	}
}

func TestWindowErrors(t *testing.T) {
	c := newCollector(t, 9, counting(def), counting(gauges))
	if rep := c.window("nope", diff, 5); rep.err == nil {
		t.Error("unknown metric: no error")
	}
	if rep := c.window("io", diff, 0); rep.err == nil {
		t.Error("zero window: no error")
	}
	if rep := c.window("io", current, 5); rep.err == nil {
		t.Error("type not accepted: no error")
	}
	if rep := c.window("memory", diff, 5); rep.err == nil {
		t.Error("type not accepted: no error")
	}
}

func TestNewErrors(t *testing.T) {
	if _, err := New(nil, 0, 10, []Metric{counting(def)}); err == nil {
		t.Error("zero interval: no error")
	}
	if _, err := New(nil, time.Second, 10, nil); err == nil {
		t.Error("no metrics: no error")
	}
	if _, err := New(nil, time.Second, 10, []Metric{counting(def), counting(def)}); err == nil {
		t.Error("duplicate metric: no error")
	}
	if _, err := New(nil, time.Second, 0, []Metric{counting(def)}); err == nil {
		t.Error("window of 0: no error")
	}
}

func TestRunAnswersRequests(t *testing.T) {
	c := newCollector(t, 9, counting(def))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()
	for {
		w, err := c.Window(ctx, "io", diff, 3)
		if err != nil {
			t.Fatal(err)
		}
		if w.Ready() && first(w.Snapshots[1]) > first(w.Snapshots[0]) {
			break
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Errorf("Run = %v, want nil after cancel", err)
	}
}

func TestRunStopsOnCollectError(t *testing.T) {
	bad := Metric{Def: def, Collect: func(*sql.DB) ([]snapshot.Entry, error) { return nil, errors.New("boom") }}
	c := newCollector(t, 9, bad)
	if err := c.Run(context.Background()); err == nil {
		t.Error("collect error: Run returned nil")
	}
}
