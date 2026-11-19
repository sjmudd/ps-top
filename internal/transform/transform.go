// Package transform turns raw query rows into snapshot rows.
package transform

import (
	"fmt"
	"unique"

	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/snapshot"
)

// Row is one raw query row: its key before mapping and its values by column index.
type Row struct {
	Key    string
	Values []uint64
}

// Aggregate maps each row's key with mapKey and merges rows that share a
// mapped key by adding their values. Output keeps first-seen key order.
func Aggregate(def metric.Definition, rows []Row, mapKey func(string) string) ([]snapshot.Row, error) {
	if err := def.Validate(); err != nil {
		return nil, err
	}
	width := len(def.Columns)
	out := make([]snapshot.Row, 0, len(rows))
	index := make(map[string]int, len(rows))
	for _, r := range rows {
		if len(r.Values) != width {
			return nil, fmt.Errorf("transform: row %q has %d values, want %d", r.Key, len(r.Values), width)
		}
		key := mapKey(r.Key)
		i, ok := index[key]
		if !ok {
			index[key] = len(out)
			out = append(out, snapshot.Row{Key: unique.Make(key), Values: append([]uint64(nil), r.Values...)})
			continue
		}
		for c, v := range r.Values {
			out[i].Values[c] += v
		}
	}
	return out, nil
}
