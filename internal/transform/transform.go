// Package transform turns raw query entries into snapshot entries, and maps
// the keys of a snapshot for a view.
package transform

import (
	"fmt"
	"unique"

	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/snapshot"
)

// Entry is one raw entry: its key before mapping and its values by column index.
type Entry struct {
	Key    string
	Values []uint64
}

// Aggregate maps each entry's key with mapKey and merges entries that share a
// mapped key, each column by its merge rule: Sum adds the values,
// DistinctPart counts the distinct non-empty values of a part of the
// entries' raw keys (before mapping). Output keeps first-seen key order. A nil
// mapKey keeps keys as they are.
func Aggregate(def metric.Definition, entries []Entry, mapKey func(string) string) ([]snapshot.Entry, error) {
	if err := def.Validate(); err != nil {
		return nil, err
	}
	if mapKey == nil {
		mapKey = func(k string) string { return k }
	}
	width := len(def.Columns)
	distinct := distinctColumns(def)
	out := make([]snapshot.Entry, 0, len(entries))
	index := make(map[string]int, len(entries))
	var seen []map[int]map[string]bool // per output entry, per distinct column: part values seen
	for _, e := range entries {
		if len(e.Values) != width {
			return nil, fmt.Errorf("transform: entry %q has %d values, want %d", e.Key, len(e.Values), width)
		}
		key := mapKey(e.Key)
		i, ok := index[key]
		if !ok {
			i = len(out)
			index[key] = i
			out = append(out, snapshot.Entry{Key: unique.Make(key), Values: make([]uint64, width)})
			if len(distinct) > 0 {
				seen = append(seen, map[int]map[string]bool{})
			}
		}
		for c, v := range e.Values {
			if def.Columns[c].Merge == metric.Sum {
				out[i].Values[c] += v
			}
		}
		if len(distinct) == 0 {
			continue
		}
		parts := snapshot.KeyParts(e.Key)
		for _, c := range distinct {
			p := part(parts, def.Columns[c].Part)
			if p == "" {
				continue
			}
			if seen[i][c] == nil {
				seen[i][c] = map[string]bool{}
			}
			seen[i][c][p] = true
			out[i].Values[c] = uint64(len(seen[i][c]))
		}
	}
	return out, nil
}

// DistinctTotals returns, for each DistinctPart column of def, the distinct
// non-empty values of its key part over all entries of raw, a snapshot
// before mapping: the column's total, which is not the sum of its values.
func DistinctTotals(def metric.Definition, raw *snapshot.Snapshot) map[int]uint64 {
	cols := distinctColumns(def)
	if len(cols) == 0 {
		return nil
	}
	seen := make(map[int]map[string]bool, len(cols))
	for _, c := range cols {
		seen[c] = map[string]bool{}
	}
	for _, e := range raw.Entries {
		parts := snapshot.KeyParts(e.Key.Value())
		for _, c := range cols {
			if p := part(parts, def.Columns[c].Part); p != "" {
				seen[c][p] = true
			}
		}
	}
	totals := make(map[int]uint64, len(cols))
	for c, s := range seen {
		totals[c] = uint64(len(s))
	}
	return totals
}

// distinctColumns returns the indexes of def's DistinctPart columns.
func distinctColumns(def metric.Definition) []int {
	var cols []int
	for c, col := range def.Columns {
		if col.Merge == metric.DistinctPart {
			cols = append(cols, c)
		}
	}
	return cols
}

// part returns parts[i], or "" if there is no such part.
func part(parts []string, i int) string {
	if i < len(parts) {
		return parts[i]
	}
	return ""
}

// Map returns s with each key mapped by mapKey and entries that share a mapped
// key merged, as Aggregate. s is not changed. A nil mapKey returns s.
func Map(def metric.Definition, s *snapshot.Snapshot, mapKey func(string) string) (*snapshot.Snapshot, error) {
	if mapKey == nil {
		return s, nil
	}
	raw := make([]Entry, len(s.Entries))
	for i, e := range s.Entries {
		raw[i] = Entry{Key: e.Key.Value(), Values: e.Values}
	}
	entries, err := Aggregate(def, raw, mapKey)
	if err != nil {
		return nil, err
	}
	return snapshot.New(s.Time, s.Width, entries)
}
