// Package transform turns raw query entries into snapshot entries.
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
// mapped key by adding their values. Output keeps first-seen key order.
func Aggregate(def metric.Definition, entries []Entry, mapKey func(string) string) ([]snapshot.Entry, error) {
	if err := def.Validate(); err != nil {
		return nil, err
	}
	width := len(def.Columns)
	out := make([]snapshot.Entry, 0, len(entries))
	index := make(map[string]int, len(entries))
	for _, e := range entries {
		if len(e.Values) != width {
			return nil, fmt.Errorf("transform: entry %q has %d values, want %d", e.Key, len(e.Values), width)
		}
		key := mapKey(e.Key)
		i, ok := index[key]
		if !ok {
			index[key] = len(out)
			out = append(out, snapshot.Entry{Key: unique.Make(key), Values: append([]uint64(nil), e.Values...)})
			continue
		}
		for c, v := range e.Values {
			out[i].Values[c] += v
		}
	}
	return out, nil
}
