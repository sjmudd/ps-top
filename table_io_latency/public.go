// Package table_io_latency contains the routines for managing table_io_waits_by_table.
package table_io_latency

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/p_s"
)

// Object contains performance_schema.table_io_waits_summary_by_table data
type Object struct {
	p_s.RelativeStats
	p_s.CollectionTime
	wantLatency bool
	initial     Rows // initial data for relative values
	current     Rows // last loaded values
	results     Rows // results (maybe with subtraction)
	totals      Row  // totals of results
	descStart   string    // start of description
}

// SetWantsLatency stores if we want to show the latency or ops from this table
func (t *Object) SetWantsLatency(wantLatency bool) {
	t.wantLatency = wantLatency
	if t.wantLatency {
		t.descStart = "Latency"
	} else {
		t.descStart = "Operations"
	}
}

// WantsLatency returns if we want to show the latency or ops from this table
func (t Object) WantsLatency() bool {
	return t.wantLatency
}

// Collect collects data from the db, updating initial values
// if needed, and then subtracting initial values if we want relative
// values, after which it stores totals.
func (t *Object) Collect(dbh *sql.DB) {
	start := time.Now()
	// lib.Logger.Println("Object.Collect() BEGIN")
	t.current = selectRows(dbh)
	lib.Logger.Println("t.current collected", len(t.current), "row(s) from SELECT")

	if len(t.initial) == 0 && len(t.current) > 0 {
		lib.Logger.Println("t.initial: copying from t.current (initial setup)")
		t.initial = make(Rows, len(t.current))
		copy(t.initial, t.current)
	}

	// check for reload initial characteristics
	if t.initial.needsRefresh(t.current) {
		lib.Logger.Println("t.initial: copying from t.current (data needs refreshing)")
		t.initial = make(Rows, len(t.current))
		copy(t.initial, t.current)
	}

	t.makeResults()

	// lib.Logger.Println( "t.initial:", t.initial )
	// lib.Logger.Println( "t.current:", t.current )
	lib.Logger.Println("t.initial.totals():", t.initial.totals())
	lib.Logger.Println("t.current.totals():", t.current.totals())
	// lib.Logger.Println("t.results:", t.results)
	// lib.Logger.Println("t.totals:", t.totals)
	lib.Logger.Println("Object.Collect() END, took:", time.Duration(time.Since(start)).String())
}

func (t *Object) makeResults() {
	// lib.Logger.Println( "- t.results set from t.current" )
	t.results = make(Rows, len(t.current))
	copy(t.results, t.current)
	if t.WantRelativeStats() {
		// lib.Logger.Println( "- subtracting t.initial from t.results as WantRelativeStats()" )
		t.results.subtract(t.initial)
	}

	// lib.Logger.Println( "- sorting t.results" )
	t.results.sort(t.wantLatency)
	// lib.Logger.Println( "- collecting t.totals from t.results" )
	t.totals = t.results.totals()
}

// SetInitialFromCurrent resets the statistics to current values
func (t *Object) SetInitialFromCurrent() {
	// lib.Logger.Println( "Object.SetInitialFromCurrent() BEGIN" )

	t.SetCollected()
	t.initial = make(Rows, len(t.current))
	copy(t.initial, t.current)

	t.makeResults()

	// lib.Logger.Println( "Object.SetInitialFromCurrent() END" )
}

// Headings returns the headings for the table
func (t Object) Headings() string {
	var r Row

	if t.wantLatency {
		return r.latencyHeadings()
	}

	return r.opsHeadings()
}

// RowContent returns the top maxRows data from the table
func (t Object) RowContent(maxRows int) []string {
	rows := make([]string, 0, maxRows)

	for i := range t.results {
		if i < maxRows {
			if t.wantLatency {
				rows = append(rows, t.results[i].latencyRowContent(t.totals))
			} else {
				rows = append(rows, t.results[i].opsRowContent(t.totals))
			}
		}
	}

	return rows
}

// EmptyRowContent returns an empty row
func (t Object) EmptyRowContent() string {
	var r Row

	if t.wantLatency {
		return r.latencyRowContent(r)
	}

	return r.opsRowContent(r)
}

// TotalRowContent returns a formated row containing totals data
func (t Object) TotalRowContent() string {
	if t.wantLatency {
		return t.totals.latencyRowContent(t.totals)
	}

	return t.totals.opsRowContent(t.totals)
}

// Description returns the description of the table as a string
func (t Object) Description() string {
	var count int
	for row := range t.results {
		if t.results[row].sumTimerWait > 0 {
			count++
		}
	}

	return fmt.Sprintf("Table %s (table_io_waits_summary_by_table) %d rows", t.descStart, count)
}

// Len returns the length of the result set
func (t Object) Len() int {
	return len(t.current)
}
