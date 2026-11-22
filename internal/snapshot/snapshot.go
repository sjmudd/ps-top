// Package snapshot holds the transformed entries returned by a single query
// taken at time Snapshot.Time, indexed by key.
package snapshot

import (
	"fmt"
	"strings"
	"time"
	"unique"
)

// Entry is one entry of a snapshot: a key and its values by column index.
// Keys are unique handles: one shared copy per distinct key, freed by the
// garbage collector when no entry refers to it; comparison is a pointer compare.
type Entry struct {
	Key    unique.Handle[string]
	Values []uint64
}

// KeySep separates the parts of a key of several columns, e.g. schema and
// table, or user, host and db. It is the ASCII unit separator, so a part may
// contain "." (as a host or a decoded table name can).
const KeySep = "\x1f"

// JoinKey returns the key of parts.
func JoinKey(parts ...string) string { return strings.Join(parts, KeySep) }

// KeyParts returns the parts of key k.
func KeyParts(k string) []string { return strings.Split(k, KeySep) }

// KeyText returns k as shown: its parts joined with ".", e.g. "shop.orders".
func KeyText(k string) string { return strings.ReplaceAll(k, KeySep, ".") }

// Snapshot is not modified after creation.
type Snapshot struct {
	Time    time.Time
	Width   int // values per entry
	Entries []Entry
	index   map[unique.Handle[string]]int
}

// New builds a snapshot from entries that each have width values. Duplicate
// keys are an error: transforms must aggregate entries that share a key.
func New(t time.Time, width int, entries []Entry) (*Snapshot, error) {
	if width < 1 {
		return nil, fmt.Errorf("snapshot: width %d, need at least 1", width)
	}
	s := &Snapshot{
		Time:    t,
		Width:   width,
		Entries: make([]Entry, len(entries)),
		index:   make(map[unique.Handle[string]]int, len(entries)),
	}
	for i, e := range entries {
		if len(e.Values) != width {
			return nil, fmt.Errorf("snapshot: entry %q has %d values, want %d", e.Key.Value(), len(e.Values), width)
		}
		if _, dup := s.index[e.Key]; dup {
			return nil, fmt.Errorf("snapshot: duplicate key %q", e.Key.Value())
		}
		s.index[e.Key] = i
		s.Entries[i] = Entry{Key: e.Key, Values: append([]uint64(nil), e.Values...)}
	}
	return s, nil
}

// Lookup returns the entry with the given key.
func (s *Snapshot) Lookup(key unique.Handle[string]) (Entry, bool) {
	i, ok := s.index[key]
	if !ok {
		return Entry{}, false
	}
	return s.Entries[i], true
}

// Total returns the sum of column col across all entries.
func (s *Snapshot) Total(col int) uint64 {
	var total uint64
	for _, e := range s.Entries {
		total += e.Values[col]
	}
	return total
}
