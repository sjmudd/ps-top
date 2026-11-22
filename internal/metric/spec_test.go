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
	grouped := s
	grouped.Where, grouped.GroupBy = "A > 0", "A, B"
	if got, want := grouped.Query(s.Tables[0]), "SELECT A, B, COUNT_READ + COUNT_WRITE AS OPS, SUM_TIMER_WAIT FROM performance_schema.some_summary WHERE A > 0 GROUP BY A, B"; got != want {
		t.Errorf("Query() with GroupBy = %q, want %q", got, want)
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
