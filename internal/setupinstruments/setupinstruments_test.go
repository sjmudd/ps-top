package setupinstruments

import (
	"context"
	"database/sql/driver"
	"errors"
	"maps"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/sjmudd/ps-top/internal/fakesql"
)

// instruments is a fake setup_instruments table: name to ENABLED, TIMED.
// failOn names an instrument whose UPDATE fails.
type instruments struct {
	mu     sync.Mutex
	rows   map[string][2]string
	failOn string
}

// handle answers Enable's SELECT and UPDATE against the table.
func (in *instruments) handle(s fakesql.Statement) fakesql.Result {
	in.mu.Lock()
	defer in.mu.Unlock()
	switch s.Query {
	case selectSQL:
		prefix := strings.TrimSuffix(s.Args[0].(string), "%")
		r := fakesql.Result{Columns: []string{"NAME", "ENABLED", "TIMED"}}
		for _, name := range slices.Sorted(maps.Keys(in.rows)) {
			v := in.rows[name]
			if strings.HasPrefix(name, prefix) && (v[0] == "NO" || v[1] == "NO") {
				r.Rows = append(r.Rows, []driver.Value{name, v[0], v[1]})
			}
		}
		return r
	case updateSQL:
		name := s.Args[2].(string)
		if name == in.failOn {
			return fakesql.Result{Err: errors.New("update refused")}
		}
		in.rows[name] = [2]string{s.Args[0].(string), s.Args[1].(string)}
		return fakesql.Result{RowsAffected: 1}
	}
	return fakesql.Result{Err: errors.New("unexpected statement: " + s.Query)}
}

func (in *instruments) get(name string) [2]string {
	in.mu.Lock()
	defer in.mu.Unlock()
	return in.rows[name]
}

// table returns instruments with one row in each state, for both patterns.
func table() *instruments {
	return &instruments{rows: map[string][2]string{
		"wait/synch/mutex/a": {"YES", "YES"},
		"wait/synch/mutex/b": {"NO", "NO"},
		"wait/synch/mutex/c": {"YES", "NO"}, // enabled but not timed
		"stage/sql/d":        {"NO", "YES"},
		"wait/io/file/e":     {"NO", "NO"}, // matches no pattern
	}}
}

func TestEnableAndRestore(t *testing.T) {
	in := table()
	db := fakesql.OpenFunc(in.handle)
	defer func() { _ = db.Close() }()

	s, err := Enable(context.Background(), db.DB, Patterns, true)
	if err != nil {
		t.Fatal(err)
	}
	if s.Changed() != 3 {
		t.Errorf("Changed() = %d, want 3 (b, c, d)", s.Changed())
	}
	for _, name := range []string{"wait/synch/mutex/a", "wait/synch/mutex/b", "wait/synch/mutex/c", "stage/sql/d"} {
		if got := in.get(name); got != [2]string{"YES", "YES"} {
			t.Errorf("after Enable %s = %v, want YES YES", name, got)
		}
	}
	if got := in.get("wait/io/file/e"); got != [2]string{"NO", "NO"} {
		t.Errorf("instrument outside the patterns changed: %v", got)
	}

	if err := s.Restore(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := table().rows
	for name, v := range want {
		if got := in.get(name); got != v {
			t.Errorf("after Restore %s = %v, want %v", name, got, v)
		}
	}
	if s.Changed() != 0 {
		t.Errorf("Changed() after Restore = %d, want 0", s.Changed())
	}
}

func TestEnableNothingToDo(t *testing.T) {
	in := &instruments{rows: map[string][2]string{"wait/synch/mutex/a": {"YES", "YES"}}}
	db := fakesql.OpenFunc(in.handle)
	defer func() { _ = db.Close() }()
	// All enabled: no UPDATE is needed, so it works even if the table
	// cannot be updated.
	s, err := Enable(context.Background(), db.DB, Patterns, false)
	if err != nil {
		t.Fatal(err)
	}
	if s.Changed() != 0 {
		t.Errorf("Changed() = %d, want 0", s.Changed())
	}
	for _, st := range db.Statements() {
		if st.Exec {
			t.Errorf("UPDATE run with nothing to change: %v", st.Args)
		}
	}
}

func TestEnableCannotUpdate(t *testing.T) {
	in := table()
	db := fakesql.OpenFunc(in.handle)
	defer func() { _ = db.Close() }()
	_, err := Enable(context.Background(), db.DB, Patterns, false)
	if err == nil || !strings.Contains(err.Error(), "'wait/synch/mutex/%' is not fully enabled and setup_instruments cannot be updated") {
		t.Fatalf("err = %v, want the pattern named", err)
	}
	for _, st := range db.Statements() {
		if st.Exec {
			t.Errorf("UPDATE run although the table cannot be updated: %v", st.Args)
		}
	}
}

// TestEnableFailureRestores checks a failed UPDATE puts back the rows already
// changed and returns the error.
func TestEnableFailureRestores(t *testing.T) {
	in := table()
	in.failOn = "stage/sql/d" // after b and c have been changed
	db := fakesql.OpenFunc(in.handle)
	defer func() { _ = db.Close() }()
	if _, err := Enable(context.Background(), db.DB, Patterns, true); err == nil || !strings.Contains(err.Error(), "enable stage/sql/d") {
		t.Fatalf("err = %v, want enable stage/sql/d to fail", err)
	}
	for name, v := range table().rows {
		if got := in.get(name); got != v {
			t.Errorf("%s = %v, want %v as before Enable", name, got, v)
		}
	}
}
