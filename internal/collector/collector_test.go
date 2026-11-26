package collector

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"sync/atomic"
	"testing"
	"time"
	"unique"

	"github.com/sjmudd/ps-top/v2/internal/metric"
	"github.com/sjmudd/ps-top/v2/internal/registry"
	"github.com/sjmudd/ps-top/v2/internal/snapshot"
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

// held returns the values of the snapshots ring name holds, oldest first.
func held(c *Collector, name string) []uint64 {
	r := c.rings[name]
	var out []uint64
	for i := range r.Len() {
		out = append(out, first(r.At(i)))
	}
	return out
}

// TestSetWindowAndReset checks SetWindow resizes every ring as New would,
// keeping the latest snapshots, and Reset keeps only the latest, both while
// the collector runs.
func TestSetWindowAndReset(t *testing.T) {
	st := new(registry.Collector)
	io := counting(def)
	io.Stats = st
	c, err := New(nil, time.Hour, 3, []Metric{io, counting(gauges)}) // no tick during the test
	if err != nil {
		t.Fatal(err)
	}
	collect(t, c, 5)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }() // collects once at start: 60
	for {
		w, err := c.Window(ctx, "io", diff, 1)
		if err != nil {
			t.Fatal(err)
		}
		if first(w.Snapshots[1]) == 60 {
			break
		}
	}
	steps := []struct {
		name   string
		change func() error
		cap    int
		io     []uint64
	}{
		{"grow to 9", func() error { return c.SetWindow(ctx, 9) }, 10, []uint64{30, 40, 50, 60}},
		{"shrink to 1", func() error { return c.SetWindow(ctx, 1) }, 2, []uint64{50, 60}},
		{"reset", func() error { return c.Reset(ctx) }, 2, []uint64{60}},
	}
	for _, s := range steps {
		if err := s.change(); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		if got := held(c, "io"); c.rings["io"].Cap() != s.cap || !slices.Equal(got, s.io) {
			t.Errorf("%s: io cap %d, held %v, want %d and %v", s.name, c.rings["io"].Cap(), got, s.cap, s.io)
		}
		if c.rings["memory"].Cap() != 1 || c.rings["memory"].Len() != 1 {
			t.Errorf("%s: memory ring cap %d, len %d, want 1 and 1", s.name, c.rings["memory"].Cap(), c.rings["memory"].Len())
		}
		if st.RingCapacity.Load() != uint64(s.cap) || st.Snapshots.Load() != uint64(len(s.io)) {
			t.Errorf("%s: CURRENT_RING_CAPACITY %d, CURRENT_SNAPSHOTS %d", s.name, st.RingCapacity.Load(), st.Snapshots.Load())
		}
	}
	if err := c.SetWindow(ctx, 0); err == nil {
		t.Error("window 0: no error")
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := c.Reset(ctx); err == nil {
		t.Error("Reset after Run ended: no error")
	}
}

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

// TestStats checks the collector records into the metric's registry entry:
// snapshots added (cumulative), snapshots held and the ring's capacity.
func TestStats(t *testing.T) {
	st := new(registry.Collector)
	m := counting(def)
	m.Stats = st
	c := newCollector(t, 3, m)
	if got := st.RingCapacity.Load(); got != 4 {
		t.Errorf("CURRENT_RING_CAPACITY = %d, want 4 (n+1)", got)
	}
	collect(t, c, 6)
	if got := st.Snapshot.Count.Load(); got != 6 {
		t.Errorf("COUNT_SNAPSHOT = %d, want 6", got)
	}
	if got := st.Snapshot.SumTimer.Load(); got == 0 {
		t.Error("SUM_TIMER_SNAPSHOT = 0, want the time taken")
	}
	if got := st.Snapshots.Load(); got != 4 {
		t.Errorf("CURRENT_SNAPSHOTS = %d, want 4 (ring full)", got)
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

// TestWindowDuringCollect checks Window answers while a collection waits on
// its query, from the snapshots already held: 10 and 20 here.
func TestWindowDuringCollect(t *testing.T) {
	io := counting(def)
	count := io.Collect
	var blocking atomic.Bool
	started, release := make(chan struct{}, 1), make(chan struct{})
	io.Collect = func(db *sql.DB) ([]snapshot.Entry, error) {
		if blocking.Load() {
			started <- struct{}{}
			<-release
		}
		return count(db)
	}
	c := newCollector(t, 9, io)
	collect(t, c, 2)
	blocking.Store(true)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()
	<-started // Run's first collection is waiting on its query

	wctx, wcancel := context.WithTimeout(ctx, time.Second)
	w, err := c.Window(wctx, "io", diff, 3)
	wcancel()
	if err != nil {
		t.Errorf("Window during a collection: %v", err)
	} else if !w.Ready() || first(w.Snapshots[0]) != 10 || first(w.Snapshots[1]) != 20 {
		t.Errorf("Window during a collection: %+v, want 10 and 20", w)
	}
	blocking.Store(false)
	close(release)
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
	if err := c.WaitForData(context.Background()); err == nil || err.Error() != "collector: io: boom" {
		t.Errorf("WaitForData after Run failed = %v, want Run's error", err)
	}
}

// TestWaitForData checks WaitForData waits for the first collection, at most
// 3 collect intervals, then returns at once, and returns when ctx is done.
func TestWaitForData(t *testing.T) {
	io := counting(def)
	count := io.Collect
	release := make(chan struct{})
	io.Collect = func(db *sql.DB) ([]snapshot.Entry, error) {
		<-release
		return count(db)
	}
	c := newCollector(t, 9, io)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()

	// The collect interval is 1ms: no data after 3ms.
	if err := c.WaitForData(ctx); err == nil || err.Error() != "collector: no data after 3ms, 3 collect intervals" {
		t.Errorf("first collection not done = %v, want the timeout", err)
	}
	done0, dcancel := context.WithCancel(ctx)
	dcancel()
	if err := c.WaitForData(done0); !errors.Is(err, context.Canceled) {
		t.Errorf("ctx done = %v, want context.Canceled", err)
	}
	close(release)
	if err := c.WaitForData(ctx); err != nil {
		t.Errorf("after the first collection = %v, want nil", err)
	}
	cancel()
	if err := <-done; err != nil {
		t.Errorf("Run = %v, want nil after cancel", err)
	}
	if err := c.WaitForData(context.Background()); err != nil {
		t.Errorf("after Run ended with data = %v, want nil", err)
	}

	// Run stopped by its ctx before any data.
	c = newCollector(t, 9, counting(def))
	stopped, scancel := context.WithCancel(context.Background())
	scancel()
	if err := c.Run(stopped); err != nil {
		t.Fatal(err)
	}
	// Run collects once before it checks ctx, so this has data: nil.
	if err := c.WaitForData(context.Background()); err != nil {
		t.Errorf("Run stopped after one collection = %v, want nil", err)
	}
}
