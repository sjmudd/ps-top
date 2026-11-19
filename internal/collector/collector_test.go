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

var def = metric.Definition{
	Name:    "io",
	Columns: []metric.Column{{Name: "sum_timer_wait", Kind: metric.Counter}},
}

// counting returns a metric whose single value is 10, 20, 30, ... per collection.
func counting() Metric {
	var v uint64
	return Metric{Def: def, Collect: func(*sql.DB) ([]snapshot.Entry, error) {
		v += 10
		return []snapshot.Entry{{Key: unique.Make("a"), Values: []uint64{v}}}, nil
	}}
}

func newCollector(t *testing.T, ringSize int, metrics ...Metric) *Collector {
	t.Helper()
	c, err := New(nil, time.Millisecond, ringSize, metrics)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func first(s *snapshot.Snapshot) uint64 { return s.Entries[0].Values[0] }

func TestWindowStartBySnapshotCount(t *testing.T) {
	c := newCollector(t, 10, counting())
	if rep := c.window("io", 5); rep.err != nil || rep.w.OK {
		t.Fatalf("no snapshots: %+v, want not OK", rep)
	}
	if err := c.collect(); err != nil {
		t.Fatal(err)
	}
	if rep := c.window("io", 5); rep.err != nil || rep.w.OK {
		t.Fatalf("one snapshot: %+v, want not OK", rep)
	}
	for range 7 {
		if err := c.collect(); err != nil {
			t.Fatal(err)
		}
	}
	// 8 snapshots held: 10 ... 80. n=5: start 30, end 80.
	rep := c.window("io", 5)
	if rep.err != nil || !rep.w.OK || first(rep.w.Start) != 30 || first(rep.w.End) != 80 {
		t.Errorf("n=5: start %d end %d (%v), want 30 and 80", first(rep.w.Start), first(rep.w.End), rep.err)
	}
	// Fewer held than asked: start is the oldest.
	rep = c.window("io", 20)
	if rep.err != nil || first(rep.w.Start) != 10 {
		t.Errorf("n=20: start %d (%v), want 10 (oldest)", first(rep.w.Start), rep.err)
	}
}

func TestWindowErrors(t *testing.T) {
	c := newCollector(t, 10, counting())
	if rep := c.window("nope", 5); rep.err == nil {
		t.Error("unknown metric: no error")
	}
	if rep := c.window("io", 0); rep.err == nil {
		t.Error("zero window: no error")
	}
}

func TestNewErrors(t *testing.T) {
	if _, err := New(nil, 0, 10, []Metric{counting()}); err == nil {
		t.Error("zero interval: no error")
	}
	if _, err := New(nil, time.Second, 10, nil); err == nil {
		t.Error("no metrics: no error")
	}
	if _, err := New(nil, time.Second, 10, []Metric{counting(), counting()}); err == nil {
		t.Error("duplicate metric: no error")
	}
	if _, err := New(nil, time.Second, 1, []Metric{counting()}); err == nil {
		t.Error("ring of 1: no error")
	}
}

func TestRunAnswersRequests(t *testing.T) {
	c := newCollector(t, 10, counting())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()
	for {
		w, err := c.Window(ctx, "io", 3)
		if err != nil {
			t.Fatal(err)
		}
		if w.OK && first(w.End) > first(w.Start) {
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
	c := newCollector(t, 10, bad)
	if err := c.Run(context.Background()); err == nil {
		t.Error("collect error: Run returned nil")
	}
}
