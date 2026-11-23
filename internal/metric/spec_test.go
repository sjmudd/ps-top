package metric

import (
	"strings"
	"testing"
)

type col int

const (
	colCount col = iota
	colWait
)

func TestSpec(t *testing.T) {
	s := Spec[col]{
		Name:   "sample",
		Tables: []string{"performance_schema.some_summary", "INFORMATION_SCHEMA.SOME_SUMMARY"},
		Keys:   []string{"A", "B"},
		Columns: []Column{
			colCount: {Name: "OPS", Kind: Counter, Expr: "COUNT_READ + COUNT_WRITE"},
			colWait:  {Name: "SUM_TIMER_WAIT", Kind: Counter},
		},
		Types:       []MetricType{DifferenceValues},
		ResetColumn: colWait,
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	if got, want := s.Query(s.Tables[1]), "SELECT A, B, COUNT_READ + COUNT_WRITE AS OPS, SUM_TIMER_WAIT FROM INFORMATION_SCHEMA.SOME_SUMMARY"; got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
	if got := s.TableName(); got != "some_summary" {
		t.Errorf("TableName() = %q, want some_summary", got)
	}
	filtered := s
	filtered.Where = "A > 0"
	if got, want := filtered.Query(s.Tables[0]), "SELECT A, B, COUNT_READ + COUNT_WRITE AS OPS, SUM_TIMER_WAIT FROM performance_schema.some_summary WHERE A > 0"; got != want {
		t.Errorf("Query() with Where = %q, want %q", got, want)
	}
	for _, bad := range []string{"some_summary", ".x", "x.", "a.b.c"} {
		b := s
		b.Tables = []string{bad}
		if err := b.Validate(); err == nil {
			t.Errorf("table %q: no error", bad)
		}
	}
	if err := s.Check(colWait); err != nil {
		t.Error(err)
	}
	if err := s.Check(col(2)); err == nil || !strings.Contains(err.Error(), "no column 2") {
		t.Errorf("Check(2) = %v", err)
	}

	// A constant left out of Columns leaves an entry with no name.
	gap := s
	gap.Columns = []Column{colWait: {Name: "SUM_TIMER_WAIT", Kind: Counter}}
	if err := gap.Validate(); err == nil {
		t.Error("gap in Columns: no error")
	}
	noKeys := s
	noKeys.Keys = nil
	if err := noKeys.Validate(); err == nil {
		t.Error("no keys: no error")
	}
}

// TestSchemaFilter checks the condition limiting a query to some schemas:
// joined with AND to a Where, else with WHERE, and none without a Schema or
// names. Schema must be a key column read by the query.
func TestSchemaFilter(t *testing.T) {
	s := Spec[col]{
		Name:        "sample",
		Tables:      []string{"performance_schema.some_summary"},
		Keys:        []string{"OBJECT_SCHEMA", "OBJECT_NAME"},
		Columns:     []Column{colCount: {Name: "COUNT_STAR", Kind: Counter}, colWait: {Name: "SUM_TIMER_WAIT", Kind: Counter}},
		Where:       "COUNT_STAR > 0",
		Schema:      "OBJECT_SCHEMA",
		Types:       []MetricType{DifferenceValues},
		ResetColumn: colWait,
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	noWhere := s
	noWhere.Where = ""
	noSchema := s
	noSchema.Schema = ""
	for _, tt := range []struct {
		spec Spec[col]
		n    int
		want string
	}{
		{s, 2, " AND OBJECT_SCHEMA IN (?,?)"},
		{s, 1, " AND OBJECT_SCHEMA IN (?)"},
		{s, 0, ""},
		{noWhere, 3, " WHERE OBJECT_SCHEMA IN (?,?,?)"},
		{noSchema, 2, ""},
	} {
		if got := tt.spec.SchemaFilter(tt.n); got != tt.want {
			t.Errorf("Where %q, Schema %q: SchemaFilter(%d) = %q, want %q", tt.spec.Where, tt.spec.Schema, tt.n, got, tt.want)
		}
	}
	notKey := s
	notKey.Schema = "DB"
	if err := notKey.Validate(); err == nil {
		t.Error("Schema not a key column: no error")
	}
	row := s
	row.Fields = []string{"OBJECT_SCHEMA", "OBJECT_NAME"}
	row.Row = func(f []string) ([]string, []uint64) { return f, []uint64{1, 1} }
	if err := row.Validate(); err == nil {
		t.Error("Schema with Row: no error")
	}
}

// TestSpecRow checks a spec whose values Row works out in Go: the query
// selects Fields, and Fields, Row, Expr and distinct parts are validated.
func TestSpecRow(t *testing.T) {
	s := Spec[col]{
		Name:   "rows",
		Tables: []string{"performance_schema.processlist"},
		Keys:   []string{"USER", "HOST"},
		Columns: []Column{
			colCount: {Name: "CONNECTIONS", Kind: Gauge},
			colWait:  {Name: "HOSTS", Kind: Gauge, Merge: DistinctPart, Part: 1},
		},
		Fields: []string{"USER", "HOST", "COMMAND"},
		Row:    func(f []string) ([]string, []uint64) { return f[:2], []uint64{1, 1} },
		Types:  []MetricType{CurrentValue},
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	if got, want := s.Query(s.Tables[0]), "SELECT USER, HOST, COMMAND FROM performance_schema.processlist"; got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
	noRow := s
	noRow.Row = nil
	if err := noRow.Validate(); err == nil {
		t.Error("Fields without Row: no error")
	}
	expr := s
	expr.Columns = []Column{colCount: {Name: "CONNECTIONS", Kind: Gauge, Expr: "COUNT(*)"}, colWait: s.Columns[colWait]}
	if err := expr.Validate(); err == nil {
		t.Error("Expr with Row: no error")
	}
	part := s
	part.Columns = []Column{colCount: s.Columns[colCount], colWait: {Name: "HOSTS", Kind: Gauge, Merge: DistinctPart, Part: 2}}
	if err := part.Validate(); err == nil {
		t.Error("distinct key part 2 of 2: no error")
	}
	counter := s
	counter.Columns = []Column{colCount: s.Columns[colCount], colWait: {Name: "HOSTS", Kind: Counter, Merge: DistinctPart, Part: 1}}
	if err := counter.Validate(); err == nil {
		t.Error("distinct counter: no error")
	}
}
