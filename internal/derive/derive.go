// Package derive computes what to display from two snapshots: the difference
// between a start and an end snapshot, and column totals.
package derive

import (
	"errors"
	"fmt"
	"time"
	"unique"

	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/snapshot"
)

// Entry is one derived entry.
type Entry struct {
	Key unique.Handle[string]
	// Values per column: counter = increase over the window;
	// gauge = current value.
	Values []uint64
	// Change per column: gauge = current minus value at window start;
	// 0 for counters.
	Change []int64
}

// Difference is the per-entry result of one metric over the period [From, To].
type Difference struct {
	// From and To are the times of the start and end snapshots.
	From, To time.Time
	// Entries has one per key of the end snapshot.
	Entries []Entry
	// Totals per column, summed over Entries' Values.
	Totals []uint64
}

// Elapsed returns the time covered by the window.
func (r Difference) Elapsed() time.Duration {
	return r.To.Sub(r.From)
}

// Window returns the difference from start to end.
//
// Rules:
//   - An entry with no value in start subtracts 0.
//   - Counters never go negative: if a value is lower than at the start, its
//     increase is 0.
//   - If the reset column's total in end is lower than in start, the source
//     was reset and its counters restarted from zero: each counter's increase
//     is its value in end.
func Window(def metric.Definition, start, end *snapshot.Snapshot) (Difference, error) {
	if err := def.Validate(); err != nil {
		return Difference{}, err
	}
	if start == nil || end == nil {
		return Difference{}, errors.New("derive: need a start and an end snapshot")
	}
	width := len(def.Columns)
	for _, s := range []*snapshot.Snapshot{start, end} {
		if s.Width != width {
			return Difference{}, fmt.Errorf("derive: metric %q has %d columns, snapshot has %d", def.Name, width, s.Width)
		}
	}
	rc := def.ResetColumn
	reset := end.Total(rc) < start.Total(rc)

	res := Difference{
		From:    start.Time,
		To:      end.Time,
		Entries: make([]Entry, len(end.Entries)),
		Totals:  make([]uint64, width),
	}
	for i, ee := range end.Entries {
		se, found := start.Lookup(ee.Key)
		e := Entry{Key: ee.Key, Values: make([]uint64, width), Change: make([]int64, width)}
		for c, col := range def.Columns {
			cur := ee.Values[c]
			var prev uint64
			if found {
				prev = se.Values[c]
			}
			switch col.Kind {
			case metric.Counter:
				if reset {
					e.Values[c] = cur
				} else {
					e.Values[c] = increase(cur, prev)
				}
			case metric.Gauge:
				e.Values[c] = cur
				e.Change[c] = int64(cur) - int64(prev)
			}
			res.Totals[c] += e.Values[c]
		}
		res.Entries[i] = e
	}
	return res, nil
}

// increase returns cur - prev, or 0 if cur is not above prev. Comparing first
// avoids uint64 wraparound.
func increase(cur, prev uint64) uint64 {
	if cur > prev {
		return cur - prev
	}
	return 0
}
