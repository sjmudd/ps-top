// Package store keeps the most recent snapshots of one metric in a ring buffer.
package store

import (
	"fmt"

	"github.com/sjmudd/ps-top/internal/snapshot"
)

// Ring holds up to a fixed number of snapshots, oldest first. Adding to a full
// ring drops the oldest snapshot.
type Ring struct {
	buf   []*snapshot.Snapshot
	start int
	n     int
}

// New returns an empty ring holding up to capacity snapshots. A ring of 1
// holds only the latest snapshot.
func New(capacity int) (*Ring, error) {
	if capacity < 1 {
		return nil, fmt.Errorf("store: capacity %d, need at least 1", capacity)
	}
	return &Ring{buf: make([]*snapshot.Snapshot, capacity)}, nil
}

// Add appends s. The ring assumes snapshots are added in time order and
// does not check it: the collector adds each snapshot right after taking
// it. A snapshot out of order would make windows wrong.
func (r *Ring) Add(s *snapshot.Snapshot) error {
	if s == nil {
		return fmt.Errorf("store: nil snapshot")
	}
	if r.n < len(r.buf) {
		r.buf[(r.start+r.n)%len(r.buf)] = s
		r.n++
		return nil
	}
	r.buf[r.start] = s
	r.start = (r.start + 1) % len(r.buf)
	return nil
}

// Len returns the number of snapshots held.
func (r *Ring) Len() int {
	return r.n
}

// At returns snapshot i, where 0 is the oldest and Len()-1 the latest.
func (r *Ring) At(i int) *snapshot.Snapshot {
	if i < 0 || i >= r.n {
		panic(fmt.Sprintf("store: index %d out of range [0,%d)", i, r.n))
	}
	return r.buf[(r.start+i)%len(r.buf)]
}

// Latest returns the most recent snapshot, or nil if the ring is empty.
func (r *Ring) Latest() *snapshot.Snapshot {
	if r.n == 0 {
		return nil
	}
	return r.At(r.n - 1)
}
