package metric

import (
	"fmt"
	"strings"
)

// Spec is the one place a metric is described: the tables it can read, which
// columns form a row's key, and its value columns. The queries and the
// Definition are all built from it, so the SQL, the stored values and the
// views cannot disagree about which column is which.
//
// Tables are tried in order at connect: the first the server can read is
// used, so a later table is a fallback, e.g. INFORMATION_SCHEMA.PROCESSLIST
// where performance_schema.processlist is missing. Every table must have the
// same columns. Write each name fully qualified and in the case the server
// uses for it: performance_schema names in lower case (upper case fails on
// MySQL and MariaDB 10.6), INFORMATION_SCHEMA names as the old program does.
//
// C is the metric's own column type: an int type with one constant per value
// column, starting at 0 in Columns order. Views name columns with these
// constants, so a view can only use columns of its own metric, and a
// misspelt column does not compile. Write Columns with the constants as
// indexes, so the order is checked too:
//
//	type Column int
//
//	const (
//		CountStar Column = iota
//		SumTimerWait
//	)
//
//	var Spec = metric.Spec[Column]{
//		Name:   "stages",
//		Tables: []string{"performance_schema.events_stages_summary_global_by_event_name"},
//		Keys:   []string{"EVENT_NAME"},
//		Columns: []metric.Column{
//			CountStar:    {Name: "COUNT_STAR", Kind: metric.Counter},
//			SumTimerWait: {Name: "SUM_TIMER_WAIT", Kind: metric.Counter},
//		},
//		Where:       "COUNT_STAR > 0",
//		Types:       []metric.MetricType{metric.DifferenceValues},
//		ResetColumn: SumTimerWait,
//	}
//
// Query(Tables[0]) then returns:
//
//	SELECT EVENT_NAME, COUNT_STAR, SUM_TIMER_WAIT FROM performance_schema.events_stages_summary_global_by_event_name WHERE COUNT_STAR > 0
//
// A constant left out of Columns leaves an entry with no name, which
// Validate reports.
type Spec[C ~int] struct {
	Name   string   // metric name, used to ask the collector for it, e.g. "stages"
	Tables []string // tables to read, fully qualified, in order of preference
	Keys   []string // key columns, read as text. A row's key is their values joined with "."
	// Columns are the value columns, read as unsigned integers:
	// Columns[c] for each constant c.
	Columns []Column
	Where   string // row filter without "WHERE", e.g. "COUNT_STAR > 0", or empty for none
	// GroupBy groups the rows without "GROUP BY", e.g. "USER", or is empty
	// for none. With it, the keys are the grouped columns and every value
	// column is an aggregate Expr, e.g. "COUNT(*)".
	GroupBy string
	// Types are the metric types views can ask for, the first being the
	// default: CurrentValue for values as they are now, DifferenceValues for
	// the change over a window.
	Types []MetricType
	// ResetColumn is the counter whose total over all rows is used to detect
	// a reset of the table. Needed only if Types has DifferenceValues.
	ResetColumn C
}

// Definition returns the value columns as a Definition, as the collector,
// transform, snapshot and derive steps use them.
func (s Spec[C]) Definition() Definition {
	return Definition{Name: s.Name, Columns: s.Columns, Types: s.Types, ResetColumn: int(s.ResetColumn)}
}

// Query returns the statement that reads the metric from table, one of
// Tables: the key columns, then the value columns in order. A column with an
// Expr is read as "Expr AS Name".
func (s Spec[C]) Query(table string) string {
	cols := append([]string(nil), s.Keys...)
	for _, c := range s.Columns {
		if c.Expr != "" {
			cols = append(cols, c.Expr+" AS "+c.Name)
			continue
		}
		cols = append(cols, c.Name)
	}
	q := "SELECT " + strings.Join(cols, ", ") + " FROM " + table
	if s.Where != "" {
		q += " WHERE " + s.Where
	}
	if s.GroupBy != "" {
		q += " GROUP BY " + s.GroupBy
	}
	return q
}

// TableName returns the first table's name without its schema, as views
// show it, e.g. "processlist".
func (s Spec[C]) TableName() string {
	if len(s.Tables) == 0 {
		return ""
	}
	t := s.Tables[0]
	return strings.ToLower(t[strings.LastIndex(t, ".")+1:])
}

// Validate checks the spec is usable.
func (s Spec[C]) Validate() error {
	if len(s.Tables) == 0 {
		return fmt.Errorf("metric %q: no table", s.Name)
	}
	for _, t := range s.Tables {
		if i := strings.Index(t, "."); i < 1 || i == len(t)-1 || strings.Count(t, ".") != 1 {
			return fmt.Errorf("metric %q: table %q is not schema.table", s.Name, t)
		}
	}
	if len(s.Keys) == 0 {
		return fmt.Errorf("metric %q: no key columns", s.Name)
	}
	for i, k := range s.Keys {
		if k == "" {
			return fmt.Errorf("metric %q: key column %d has no name", s.Name, i)
		}
	}
	return s.Definition().Validate()
}

// Check returns an error if c is not one of the spec's columns: a constant
// added without its entry in Columns.
func (s Spec[C]) Check(c C) error {
	if c < 0 || int(c) >= len(s.Columns) {
		return fmt.Errorf("metric %q: no column %d", s.Name, int(c))
	}
	return nil
}
