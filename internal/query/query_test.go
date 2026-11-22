package query

import (
	"database/sql/driver"
	"testing"

	"github.com/sjmudd/ps-top/internal/fakesql"
)

func TestRunInvalidType(t *testing.T) {
	// The type check runs before the database is used, so a nil db is safe.
	if err := Run(nil, "SELECT 1", []Column{{"a", Type(9)}}, func(Row) {}); err == nil {
		t.Error("invalid type: no error")
	}
}

// TestRunScans checks each column is scanned by its type, NULL included.
func TestRunScans(t *testing.T) {
	db := fakesql.Open(fakesql.Result{
		Columns: []string{"NAME", "COUNT_STAR"},
		Rows: [][]driver.Value{
			{"a", int64(7)},
			{nil, nil},
		},
	})
	defer func() { _ = db.Close() }()
	type got struct {
		name         string
		count        uint64
		nullS, nullU bool
	}
	var rows []got
	err := Run(db.DB, "SELECT NAME, COUNT_STAR FROM t", []Column{{"NAME", String}, {"COUNT_STAR", Uint}}, func(r Row) {
		rows = append(rows, got{r.Str(0), r.Uint(1), r.Null(0), r.Null(1)})
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []got{{"a", 7, false, false}, {"", 0, true, true}}
	if len(rows) != len(want) || rows[0] != want[0] || rows[1] != want[1] {
		t.Errorf("rows = %+v, want %+v", rows, want)
	}
}

// TestRunColumnCount checks a result with another number of columns fails.
func TestRunColumnCount(t *testing.T) {
	db := fakesql.Open(fakesql.Result{Columns: []string{"NAME"}, Rows: [][]driver.Value{{"a"}}})
	defer func() { _ = db.Close() }()
	if err := Run(db.DB, "SELECT NAME FROM t", []Column{{"NAME", String}, {"COUNT_STAR", Uint}}, func(Row) {}); err == nil {
		t.Error("1 column, want 2: no error")
	}
}
