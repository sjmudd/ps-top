// Package derive computes what to display from stored snapshots: the difference
// over a sliding window and column totals.
package derive

import (
	"fmt"
	"time"
	"unique"

	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/store"
)

// Row is one derived row.
type Row struct {
	Key unique.Handle[string]
	// Values per column: counter = increase over the window;
	// gauge = current value.
	Values []uint64
	// Change per column: gauge = current minus value at window start;
	// 0 for counters.
	Change []int64
}

// Difference is the per-row result of one metric over the period [From, To].
type Difference struct {
	// Ready is false until two snapshots exist; Rows is then empty.
	Ready bool
	// From and To are the times of the start and end snapshots used.
	From, To time.Time
	// Reset is true if a reset moved the window start forward.
	Reset bool
	// Rows has one entry per row of the latest snapshot.
	Rows []Row
	// Totals per column, summed over Rows' Values.
	Totals []uint64
}

// Elapsed returns the time covered by the window.
func (r Difference) Elapsed() time.Duration {
	return r.To.Sub(r.From)
}

// Window returns the difference between the latest snapshot in ring and the
// snapshot n before it (the oldest held if fewer are held).
//
// Rules:
//   - A row with no earlier value at the window start subtracts 0.
//   - Counters never go negative: if a value is lower than at the start, its
//     increase is 0.
//   - If the reset column's total is lower than at the start, the source was
//     reset: the start moves forward to the earliest snapshot whose total is
//     not above the latest total. This may be the latest snapshot itself,
//     giving a zero-length window and zero increases.
func Window(def metric.Definition, ring *store.Ring, n int) (Difference, error) {
	if err := def.Validate(); err != nil {
		return Difference{}, err
	}
	if n < 1 {
		return Difference{}, fmt.Errorf("derive: window of %d snapshots, need at least 1", n)
	}
	held := ring.Len()
	if held < 2 {
		return Difference{}, nil
	}
	width := len(def.Columns)
	latest := ring.At(held - 1)
	if latest.Width != width {
		return Difference{}, fmt.Errorf("derive: metric %q has %d columns, snapshot has %d", def.Name, width, latest.Width)
	}

	start := max(held-1-n, 0)
	rc := def.ResetColumn
	latestTotal := latest.Total(rc)
	reset := false
	if latestTotal < ring.At(start).Total(rc) {
		reset = true
		for j := start + 1; j < held; j++ {
			if ring.At(j).Total(rc) <= latestTotal {
				start = j
				break
			}
		}
	}
	old := ring.At(start)
	if old.Width != width {
		return Difference{}, fmt.Errorf("derive: metric %q has %d columns, snapshot has %d", def.Name, width, old.Width)
	}

	res := Difference{
		Ready:  true,
		From:   old.Time,
		To:     latest.Time,
		Reset:  reset,
		Rows:   make([]Row, len(latest.Rows)),
		Totals: make([]uint64, width),
	}
	for i, lr := range latest.Rows {
		or, found := old.Lookup(lr.Key)
		row := Row{Key: lr.Key, Values: make([]uint64, width), Change: make([]int64, width)}
		for c, col := range def.Columns {
			cur := lr.Values[c]
			var prev uint64
			if found {
				prev = or.Values[c]
			}
			switch col.Kind {
			case metric.Counter:
				row.Values[c] = increase(cur, prev)
			case metric.Gauge:
				row.Values[c] = cur
				row.Change[c] = int64(cur) - int64(prev)
			}
			res.Totals[c] += row.Values[c]
		}
		res.Rows[i] = row
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
