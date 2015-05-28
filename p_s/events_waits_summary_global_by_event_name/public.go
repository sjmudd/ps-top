// Package events_waits_summary_global_by_event_name provides library routines for ps-top.
// for managing the events_waits_summary_global_by_event_name table.
package events_waits_summary_global_by_event_name

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/p_s"
)

// Object holds a table of rows
type Object struct {
	p_s.RelativeStats
	p_s.CollectionTime
	wantLatency bool
	initial      Rows // initial data for relative values
	current      Rows // last loaded values
	results      Rows // results (maybe with subtraction)
	totals       Row  // totals of results
}

// SetWantsLatency allows us to define if we want latency settings
func (t *Object) SetWantsLatency(wantLatency bool) {
	t.wantLatency = wantLatency
}

// WantsLatency returns whether we want to see latency information
func (t Object) WantsLatency() bool {
	return t.wantLatency
}

// Collect collects data from the db, updating initial
// values if needed, and then subtracting initial values if we want
// relative values, after which it stores totals.
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
	t.results.sort()
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

// EmptyRowContent returns a string representation of no data
func (t Object) EmptyRowContent() string {
	return t.emptyRowContent()
}

// Headings returns a string representation of the headings
func (t *Object) Headings() string {
	var r Row

	return r.headings()
}

// RowContent returns a string representation of the row content
func (t Object) RowContent(maxRows int) []string {
	rows := make([]string, 0, maxRows)

	for i := range t.results {
		if i < maxRows {
			rows = append(rows, t.results[i].rowContent(t.totals))
		}
	}

	return rows
}

func (t Object) emptyRowContent() string {
	var r Row

	return r.rowContent(r)
}

// TotalRowContent returns a string representation of the totals of the table
func (t Object) TotalRowContent() string {
	return t.totals.rowContent(t.totals)
}

// Description returns a description of the table
func (t Object) Description() string {
	var count int
	for row := range t.results {
		if t.results[row].SUM_TIMER_WAIT > 0 {
			count++
		}
	}
	return fmt.Sprintf("Mutex Latency (events_waits_summary_global_by_event_name) %d rows", count)
}

// Len returns the length of the result set
func (t Object) Len() int {
	return len(t.results)
}
