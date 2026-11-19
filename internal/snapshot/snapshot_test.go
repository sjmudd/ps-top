package snapshot

import (
	"strings"
	"testing"
	"time"
	"unique"
)

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func k(s string) unique.Handle[string] { return unique.Make(s) }

func TestNewLookupTotal(t *testing.T) {
	s, err := New(t0, 2, []Row{
		{Key: k("db.a"), Values: []uint64{1, 10}},
		{Key: k("db.b"), Values: []uint64{2, 20}},
	})
	if err != nil {
		t.Fatal(err)
	}
	r, ok := s.Lookup(k("db.b"))
	if !ok || r.Values[1] != 20 {
		t.Errorf("Lookup(db.b) = %v, %v", r, ok)
	}
	if _, ok := s.Lookup(k("db.c")); ok {
		t.Error("Lookup(db.c) found a row")
	}
	if got := s.Total(1); got != 30 {
		t.Errorf("Total(1) = %d, want 30", got)
	}
}

func TestNewErrors(t *testing.T) {
	if _, err := New(t0, 0, nil); err == nil {
		t.Error("width 0: no error")
	}
	if _, err := New(t0, 2, []Row{{Key: k("a"), Values: []uint64{1}}}); err == nil {
		t.Error("wrong width: no error")
	}
	if _, err := New(t0, 1, []Row{{Key: k("a"), Values: []uint64{1}}, {Key: k("a"), Values: []uint64{2}}}); err == nil {
		t.Error("duplicate key: no error")
	}
}

func TestNewCopiesValues(t *testing.T) {
	in := []Row{{Key: k("a"), Values: []uint64{1}}}
	s, err := New(t0, 1, in)
	if err != nil {
		t.Fatal(err)
	}
	in[0].Values[0] = 99
	if s.Rows[0].Values[0] != 1 {
		t.Error("snapshot shares the caller's values slice")
	}
}

func TestLookupAcrossSnapshots(t *testing.T) {
	// Equal keys with separate backing memory, as each snapshot's query would give.
	s1, err := New(t0, 1, []Row{{Key: k(strings.Clone("/var/lib/mysql/db/t1.ibd")), Values: []uint64{1}}})
	if err != nil {
		t.Fatal(err)
	}
	s2, err := New(t0.Add(time.Second), 1, []Row{{Key: k(strings.Clone("/var/lib/mysql/db/t1.ibd")), Values: []uint64{2}}})
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := s1.Lookup(s2.Rows[0].Key); !ok || r.Values[0] != 1 {
		t.Errorf("key from s2 in s1: %v, %v", r, ok)
	}
}
