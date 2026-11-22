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
	"strings"

	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/metrics/fileio"
	"github.com/sjmudd/ps-top/internal/metrics/memory"
	"github.com/sjmudd/ps-top/internal/metrics/stages"
	"github.com/sjmudd/ps-top/internal/metrics/synchwaits"
	"github.com/sjmudd/ps-top/internal/metrics/tableio"
	"github.com/sjmudd/ps-top/internal/metrics/tablelocks"
	"github.com/sjmudd/ps-top/internal/probe"
	"github.com/sjmudd/ps-top/internal/query"
	"github.com/sjmudd/ps-top/internal/snapshot"
	"github.com/sjmudd/ps-top/internal/transform"
)

// Metric is a metric as the probe and the collector use it, built from a
// Spec by From.
type Metric struct {
	Def     metric.Definition
	Queries []probe.Candidate // the first usable one is chosen at connect
	columns []query.Column    // result columns: keys, then values
	keys    int               // number of key columns
}

// From builds the Metric of s.
func From[C ~int](s metric.Spec[C]) Metric {
	m := Metric{
		Def:     s.Definition(),
		Queries: []probe.Candidate{{Name: s.Table, Query: s.Query()}},
		keys:    len(s.Keys),
	}
	for _, k := range s.Keys {
		m.columns = append(m.columns, query.Column{Name: k, Type: query.String})
	}
	for _, c := range s.Columns {
		m.columns = append(m.columns, query.Column{Name: c.Name, Type: query.Uint})
	}
	return m
}

// Collect runs statement, one of m.Queries, and returns the entries for one
// snapshot. A row's key is its key columns joined with ".". Rows with the
// same key are merged.
func (m Metric) Collect(db *sql.DB, statement string) ([]snapshot.Entry, error) {
	var raw []transform.Entry
	err := query.Run(db, statement, m.columns, func(r query.Row) {
		parts := make([]string, m.keys)
		for i := range parts {
			parts[i] = r.Str(i)
		}
		values := make([]uint64, len(m.columns)-m.keys)
		for i := range values {
			values[i] = r.Uint(m.keys + i)
		}
		raw = append(raw, transform.Entry{Key: strings.Join(parts, "."), Values: values})
	})
	if err != nil {
		return nil, err
	}
	return transform.Aggregate(m.Def, raw, nil)
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
	}
}
