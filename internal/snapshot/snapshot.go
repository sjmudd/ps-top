// Package snapshot holds the transformed rows returned by a single query taken
// at time Snapshot.Time, indexed by key.
package snapshot

import (
	"fmt"
	"time"
	"unique"
)

// Row is one row of a snapshot: a key and its values by column index.
// Keys are unique handles: one shared copy per distinct key, freed by the
// garbage collector when no row refers to it; comparison is a pointer compare.
type Row struct {
	Key    unique.Handle[string]
	Values []uint64
}

// Snapshot is not modified after creation.
type Snapshot struct {
	Time  time.Time
	Width int // values per row
	Rows  []Row
	index map[unique.Handle[string]]int
}

// New builds a snapshot from rows that each have width values. Duplicate keys
// are an error: transforms must aggregate rows that share a key.
func New(t time.Time, width int, rows []Row) (*Snapshot, error) {
	if width < 1 {
		return nil, fmt.Errorf("snapshot: width %d, need at least 1", width)
	}
	s := &Snapshot{
		Time:  t,
		Width: width,
		Rows:  make([]Row, len(rows)),
		index: make(map[unique.Handle[string]]int, len(rows)),
	}
	for i, r := range rows {
		if len(r.Values) != width {
			return nil, fmt.Errorf("snapshot: row %q has %d values, want %d", r.Key.Value(), len(r.Values), width)
		}
		if _, dup := s.index[r.Key]; dup {
			return nil, fmt.Errorf("snapshot: duplicate key %q", r.Key.Value())
		}
		s.index[r.Key] = i
		s.Rows[i] = Row{Key: r.Key, Values: append([]uint64(nil), r.Values...)}
	}
	return s, nil
}

// Lookup returns the row with the given key.
func (s *Snapshot) Lookup(key unique.Handle[string]) (Row, bool) {
	i, ok := s.index[key]
	if !ok {
		return Row{}, false
	}
	return s.Rows[i], true
}

// Total returns the sum of column col across all rows.
func (s *Snapshot) Total(col int) uint64 {
	var total uint64
	for _, r := range s.Rows {
		total += r.Values[col]
	}
	return total
}
