// Package metrics holds the metric definitions: for each metric, the queries
// that can read it, its result columns and how a result row becomes an entry.
// Snapshots keep the raw keys; views map them.
package metrics

import (
	"database/sql"

	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/probe"
	"github.com/sjmudd/ps-top/internal/query"
	"github.com/sjmudd/ps-top/internal/snapshot"
	"github.com/sjmudd/ps-top/internal/transform"
)

// Metric is one metric and how to read it.
type Metric struct {
	Def     metric.Definition
	Queries []probe.Candidate // the first usable one is chosen at connect
	Columns []query.Column    // result columns of every candidate query
	Entry   func(query.Row) transform.Entry
}

// Collect runs statement and returns the entries for one snapshot. Rows with
// the same raw key are merged.
func (m Metric) Collect(db *sql.DB, statement string) ([]snapshot.Entry, error) {
	var raw []transform.Entry
	if err := query.Run(db, statement, m.Columns, func(r query.Row) { raw = append(raw, m.Entry(r)) }); err != nil {
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
	return []Metric{TableIO, FileIO, SynchWaits, Stages, TableLocks}
}
