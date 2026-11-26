package store

import (
	"slices"
	"testing"
	"time"

	"github.com/sjmudd/ps-top/v2/internal/snapshot"
)

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func snap(t *testing.T, sec int) *snapshot.Snapshot {
	t.Helper()
	s, err := snapshot.New(t0.Add(time.Duration(sec)*time.Second), 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNewCapacity(t *testing.T) {
	if _, err := New(0); err == nil {
		t.Error("capacity 0: no error")
	}
	r, err := New(3)
	if err != nil {
		t.Fatal(err)
	}
	if r.Len() != 0 || r.Latest() != nil {
		t.Errorf("new ring: Len=%d Latest=%v", r.Len(), r.Latest())
	}
}

func TestAddWraps(t *testing.T) {
	r, err := New(3)
	if err != nil {
		t.Fatal(err)
	}
	for sec := 0; sec < 5; sec++ {
		if err := r.Add(snap(t, sec)); err != nil {
			t.Fatal(err)
		}
	}
	if r.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", r.Len())
	}
	for i, wantSec := range []int{2, 3, 4} {
		if got := r.At(i).Time; !got.Equal(t0.Add(time.Duration(wantSec) * time.Second)) {
			t.Errorf("At(%d).Time = %v, want +%ds", i, got, wantSec)
		}
	}
	if !r.Latest().Time.Equal(t0.Add(4 * time.Second)) {
		t.Errorf("Latest().Time = %v", r.Latest().Time)
	}
}

// seconds returns the times of the snapshots held, as seconds after t0,
// oldest first.
func seconds(r *Ring) []int {
	var out []int
	for i := range r.Len() {
		out = append(out, int(r.At(i).Time.Sub(t0)/time.Second))
	}
	return out
}

// wrapped returns a ring of capacity 3 holding snapshots 2, 3 and 4, its
// start moved by wrapping.
func wrapped(t *testing.T) *Ring {
	t.Helper()
	r, err := New(3)
	if err != nil {
		t.Fatal(err)
	}
	for sec := range 5 {
		if err := r.Add(snap(t, sec)); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

// TestResize checks growing keeps every snapshot in order and adds room,
// and shrinking keeps the latest.
func TestResize(t *testing.T) {
	r := wrapped(t)
	if err := r.Resize(5); err != nil {
		t.Fatal(err)
	}
	if err := r.Add(snap(t, 5)); err != nil {
		t.Fatal(err)
	}
	if got := seconds(r); r.Cap() != 5 || !slices.Equal(got, []int{2, 3, 4, 5}) {
		t.Errorf("grown: cap %d, held %v, want 5 and [2 3 4 5]", r.Cap(), got)
	}
	if err := r.Resize(2); err != nil {
		t.Fatal(err)
	}
	if got := seconds(r); r.Cap() != 2 || !slices.Equal(got, []int{4, 5}) {
		t.Errorf("shrunk: cap %d, held %v, want 2 and [4 5]", r.Cap(), got)
	}
	if err := r.Resize(0); err == nil {
		t.Error("capacity 0: no error")
	}
}

// TestTrim checks Trim keeps the latest snapshots, and the ring fills
// again after it.
func TestTrim(t *testing.T) {
	r := wrapped(t)
	r.Trim(1)
	if got := seconds(r); !slices.Equal(got, []int{4}) {
		t.Fatalf("trimmed to 1: %v, want [4]", got)
	}
	for sec := 5; sec < 8; sec++ {
		if err := r.Add(snap(t, sec)); err != nil {
			t.Fatal(err)
		}
	}
	if got := seconds(r); !slices.Equal(got, []int{5, 6, 7}) {
		t.Errorf("after adding: %v, want [5 6 7]", got)
	}
	r.Trim(5)
	if r.Len() != 3 {
		t.Errorf("Trim above Len: %d held, want 3", r.Len())
	}
	r.Trim(0)
	if r.Len() != 0 || r.Latest() != nil {
		t.Errorf("Trim(0): %d held", r.Len())
	}
}

func TestAddRejects(t *testing.T) {
	r, err := New(2)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Add(nil); err == nil {
		t.Error("nil snapshot: no error")
	}
}

func TestAtPanics(t *testing.T) {
	r, err := New(2)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Error("At(0) on empty ring did not panic")
		}
	}()
	r.At(0)
}
