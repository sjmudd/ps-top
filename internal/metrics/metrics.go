// Package metrics lists the metrics the program collects, and reads one
// snapshot of a metric from the server.
//
// Each metric has its own package under internal/metrics, e.g. tableio. That
// package holds the metric's column type, one constant per value column, and
// its metric.Spec: the table, key columns, value columns and row filter. The
// query, the stored values and the views are all built from the Spec.
//
// # How the parts work together
//
// For the stages metric, from server to screen:
//
//	stages.Spec                   one place: table, keys, value columns, filter
//	  -> Spec.Query()             SELECT EVENT_NAME, COUNT_STAR, SUM_TIMER_WAIT FROM ...
//	  -> probe                    checks the query runs on this server
//	  -> Metric.Collect           one row per result row: key "stage/sql/Sending data",
//	                              values [COUNT_STAR, SUM_TIMER_WAIT] in Spec.Columns order
//	  -> collector                keeps the last snapshots of each metric, by Spec.Name,
//	                              and sends those the metric type needs: 2 for
//	                              DifferenceValues (window start and end), 1 for CurrentValue
//	  -> presenter.View           names columns with stages.SumTimerWait etc.,
//	                              maps keys, and draws the difference over the window
//	                              (or the current values)
//
// Snapshots keep the raw keys. Views map them, so several views can share one
// metric.
//
// # Adding a metric and a view
//
// What is needed, and where it goes:
//
//  1. The performance_schema table, the columns that identify a row (the
//     key) and the value columns to keep, with each value's kind (a Counter
//     only goes up, a Gauge goes up and down). Pick the counter used to
//     detect a reset, usually SUM_TIMER_WAIT, and the row filter, usually
//     COUNT_STAR > 0 so rows with no activity are not read. Decide the metric
//     types: DifferenceValues to show the change over a window (needs the
//     reset counter), CurrentValue to show values as they are now, e.g.
//     memory in use.
//  2. A package internal/metrics/<name> with type Column int, one constant
//     per value column and var Spec (see metric.Spec for an example).
//  3. Its Spec in All below. The probe then checks its query at connect, and
//     the collector reads it. If the server cannot run the query, e.g. the
//     table is missing, the metric and its views are left out.
//  4. A view in internal/presenter: a title, the Spec, how to map raw keys
//     (or nil), the cells of a line, and the column that orders the rows
//     (see the presenter package).
//  5. The view in the app's list of views.
package metrics

import (
	"database/sql"
	"fmt"

	"github.com/sjmudd/ps-top/v2/internal/metric"
	"github.com/sjmudd/ps-top/v2/internal/metrics/fileio"
	"github.com/sjmudd/ps-top/v2/internal/metrics/memory"
	"github.com/sjmudd/ps-top/v2/internal/metrics/stages"
	"github.com/sjmudd/ps-top/v2/internal/metrics/synchwaits"
	"github.com/sjmudd/ps-top/v2/internal/metrics/tableio"
	"github.com/sjmudd/ps-top/v2/internal/metrics/tablelocks"
	"github.com/sjmudd/ps-top/v2/internal/metrics/userlatency"
	"github.com/sjmudd/ps-top/v2/internal/probe"
	"github.com/sjmudd/ps-top/v2/internal/query"
	"github.com/sjmudd/ps-top/v2/internal/registry"
	"github.com/sjmudd/ps-top/v2/internal/snapshot"
	"github.com/sjmudd/ps-top/v2/internal/transform"
)

// Metric is a metric as the probe and the collector use it, built from a
// Spec by From.
type Metric struct {
	Def     metric.Definition
	Queries []probe.Candidate // the first usable one is chosen at connect
	columns []query.Column    // result columns: keys, then values, or the Spec's Fields
	keys    int               // number of key columns, or key parts Row returns
	// row is the Spec's Row: with it, columns are text fields and row works
	// out each row's key parts and values.
	row func([]string) ([]string, []uint64)
	// schemaFilter is the Spec's SchemaFilter: what limits a query to some
	// schemas.
	schemaFilter func(n int) string
}

// From builds the Metric of s.
func From[C ~int](s metric.Spec[C]) Metric {
	m := Metric{Def: s.Definition(), keys: len(s.Keys), row: s.Row, schemaFilter: s.SchemaFilter}
	for _, t := range s.Tables {
		m.Queries = append(m.Queries, probe.Candidate{Name: t, Query: s.Query(t)})
	}
	if s.Row != nil {
		for _, f := range s.Fields {
			m.columns = append(m.columns, query.Column{Name: f, Type: query.String})
		}
		return m
	}
	for _, k := range s.Keys {
		m.columns = append(m.columns, query.Column{Name: k, Type: query.String})
	}
	for _, c := range s.Columns {
		m.columns = append(m.columns, query.Column{Name: c.Name, Type: query.Uint})
	}
	return m
}

// Columns returns the result columns of m's queries, keys first, then values.
func (m Metric) Columns() []query.Column { return append([]query.Column(nil), m.columns...) }

// Collect runs statement, one of m.Queries, and returns the entries for one
// snapshot. A row's key is its key columns as parts (snapshot.JoinKey), or
// the key parts the Spec's Row returns. Rows with the same key are merged.
// The query and the merge are recorded in st.
//
// With filteredSchemas given (--database-filter, nil: no filter), a metric
// whose Spec has a Schema reads only the rows of those schemas, as the old
// program did, e.g. "... WHERE COUNT_STAR > 0 AND OBJECT_SCHEMA IN (?,?)"
// with the names as arguments. Other metrics read every row.
func (m Metric) Collect(db *sql.DB, statement string, filteredSchemas []string, st *registry.Collector) ([]snapshot.Entry, error) {
	var raw []transform.Entry
	var rowErr error
	condition := ""
	if m.schemaFilter != nil {
		condition = m.schemaFilter(len(filteredSchemas))
	}
	if condition == "" {
		filteredSchemas = nil // a metric with no schema column reads every row
	}
	statement += condition
	t := st.Query.Start()
	err := query.Run(db, statement, filteredSchemas, m.columns, func(r query.Row) {
		if m.row != nil {
			fields := make([]string, len(m.columns))
			for i := range fields {
				fields[i] = r.Str(i)
			}
			parts, values := m.row(fields)
			if (len(parts) != m.keys || len(values) != len(m.Def.Columns)) && rowErr == nil {
				rowErr = fmt.Errorf("metrics: %s: row gave %d key parts and %d values, want %d and %d",
					m.Def.Name, len(parts), len(values), m.keys, len(m.Def.Columns))
			}
			raw = append(raw, transform.Entry{Key: snapshot.JoinKey(parts...), Values: values})
			return
		}
		parts := make([]string, m.keys)
		for i := range parts {
			parts[i] = r.Str(i)
		}
		values := make([]uint64, len(m.columns)-m.keys)
		for i := range values {
			values[i] = r.Uint(m.keys + i)
		}
		raw = append(raw, transform.Entry{Key: snapshot.JoinKey(parts...), Values: values})
	})
	t.Stop()
	if err == nil {
		err = rowErr
	}
	st.RowsRead.Add(uint64(len(raw)))
	if err != nil {
		st.QueryErrors.Add(1)
		return nil, err
	}
	t = st.Merge.Start()
	entries, err := transform.Aggregate(m.Def, raw, nil)
	t.Stop()
	st.Entries.Add(uint64(len(entries)))
	return entries, err
}

// Request returns what the probe must check for all metrics: every candidate
// query.
func Request() probe.Request {
	var req probe.Request
	for _, m := range All() {
		req.Candidates = append(req.Candidates, m.Queries...)
	}
	return req
}

// All returns every metric.
func All() []Metric {
	return []Metric{
		From(tableio.Spec),
		From(fileio.Spec),
		From(synchwaits.Spec),
		From(stages.Spec),
		From(tablelocks.Spec),
		From(memory.Spec),
		From(userlatency.Spec),
	}
}
