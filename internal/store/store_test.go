package store

import (
	"testing"
	"time"

	"github.com/sjmudd/ps-top/internal/snapshot"
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
	if _, err := New(1); err == nil {
		t.Error("capacity 1: no error")
	}
	r, err := New(3)
	if err != nil {
		t.Fatal(err)
	}
	if r.Cap() != 3 || r.Len() != 0 || r.Latest() != nil {
		t.Errorf("new ring: Cap=%d Len=%d Latest=%v", r.Cap(), r.Len(), r.Latest())
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

func TestAddRejects(t *testing.T) {
	r, err := New(2)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Add(nil); err == nil {
		t.Error("nil snapshot: no error")
	}
	if err := r.Add(snap(t, 5)); err != nil {
		t.Fatal(err)
	}
	if err := r.Add(snap(t, 4)); err == nil {
		t.Error("older snapshot: no error")
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
