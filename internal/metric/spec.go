package metric

import (
	"fmt"
	"strings"
)

// Spec is the one place a metric is described: the performance_schema table
// it reads, which columns form a row's key, and its value columns. The query
// and the Definition are both built from it, so the SQL, the stored values
// and the views cannot disagree about which column is which.
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
//		Name:  "stages",
//		Table: "events_stages_summary_global_by_event_name",
//		Keys:  []string{"EVENT_NAME"},
//		Columns: []metric.Column{
//			CountStar:    {Name: "COUNT_STAR", Kind: metric.Counter},
//			SumTimerWait: {Name: "SUM_TIMER_WAIT", Kind: metric.Counter},
//		},
//		Where:       "COUNT_STAR > 0",
//		Types:       []metric.MetricType{metric.DifferenceValues},
//		ResetColumn: SumTimerWait,
//	}
//
// Query then returns:
//
//	SELECT EVENT_NAME, COUNT_STAR, SUM_TIMER_WAIT FROM performance_schema.events_stages_summary_global_by_event_name WHERE COUNT_STAR > 0
//
// A constant left out of Columns leaves an entry with no name, which
// Validate reports.
type Spec[C ~int] struct {
	Name  string   // metric name, used to ask the collector for it, e.g. "stages"
	Table string   // table in performance_schema
	Keys  []string // key columns, read as text. A row's key is their values joined with "."
	// Columns are the value columns, read as unsigned integers:
	// Columns[c] for each constant c.
	Columns []Column
	Where   string // row filter without "WHERE", e.g. "COUNT_STAR > 0", or empty for none
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

// Query returns the statement that reads the metric: the key columns, then
// the value columns in order. A column with an Expr is read as "Expr AS Name".
func (s Spec[C]) Query() string {
	cols := append([]string(nil), s.Keys...)
	for _, c := range s.Columns {
		if c.Expr != "" {
			cols = append(cols, c.Expr+" AS "+c.Name)
			continue
		}
		cols = append(cols, c.Name)
	}
	q := "SELECT " + strings.Join(cols, ", ") + " FROM performance_schema." + s.Table
	if s.Where != "" {
		q += " WHERE " + s.Where
	}
	return q
}

// Validate checks the spec is usable.
func (s Spec[C]) Validate() error {
	if s.Table == "" {
		return fmt.Errorf("metric %q: no table", s.Name)
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
