// Package mutex_latency provides library routines for ps-top.
// for managing the events_waits_summary_global_by_event_name table.
package mutex_latency

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sjmudd/ps-top/baseobject"
	"github.com/sjmudd/ps-top/logger"
)

// Object holds a table of rows
type Object struct {
	baseobject.BaseObject      // embedded
	initial               Rows // initial data for relative values
	current               Rows // last loaded values
	results               Rows // results (maybe with subtraction)
	totals                Row  // totals of results
}

func (t *Object) copyCurrentToInitial() {
	t.initial = make(Rows, len(t.current))
	t.SetInitialCollectTime(t.LastCollectTime())
	copy(t.initial, t.current)
}

// Collect collects data from the db, updating initial
// values if needed, and then subtracting initial values if we want
// relative values, after which it stores totals.
func (t *Object) Collect(dbh *sql.DB) {
	start := time.Now()
	// logger.Println("Object.Collect() BEGIN")
	t.current = selectRows(dbh)
	t.SetLastCollectTimeNow()

	logger.Println("t.current collected", len(t.current), "row(s) from SELECT")

	if len(t.initial) == 0 && len(t.current) > 0 {
		logger.Println("t.initial: copying from t.current (initial setup)")
		t.copyCurrentToInitial()
	}

	// check for reload initial characteristics
	if t.initial.needsRefresh(t.current) {
		logger.Println("t.initial: copying from t.current (data needs refreshing)")
		t.copyCurrentToInitial()
	}

	t.makeResults()

	// logger.Println( "t.initial:", t.initial )
	// logger.Println( "t.current:", t.current )
	logger.Println("t.initial.totals():", t.initial.totals())
	logger.Println("t.current.totals():", t.current.totals())
	// logger.Println("t.results:", t.results)
	// logger.Println("t.totals:", t.totals)
	logger.Println("Object.Collect() END, took:", time.Duration(time.Since(start)).String())
}

func (t *Object) makeResults() {
	// logger.Println( "- t.results set from t.current" )
	t.results = make(Rows, len(t.current))
	copy(t.results, t.current)
	if t.WantRelativeStats() {
		// logger.Println( "- subtracting t.initial from t.results as WantRelativeStats()" )
		t.results.subtract(t.initial)
	}

	// logger.Println( "- sorting t.results" )
	t.results.sort()
	// logger.Println( "- collecting t.totals from t.results" )
	t.totals = t.results.totals()
}

// SetInitialFromCurrent resets the statistics to current values
func (t *Object) SetInitialFromCurrent() {
	// logger.Println( "Object.SetInitialFromCurrent() BEGIN" )

	t.copyCurrentToInitial()

	t.makeResults()

	// logger.Println( "Object.SetInitialFromCurrent() END" )
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
func (t Object) RowContent() []string {
	rows := make([]string, 0, len(t.results))

	for i := range t.results {
		rows = append(rows, t.results[i].rowContent(t.totals))
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
		if t.results[row].sumTimerWait > 0 {
			count++
		}
	}
	return fmt.Sprintf("Mutex Latency (events_waits_summary_global_by_event_name) %d rows", count)
}

// Len returns the length of the result set
func (t Object) Len() int {
	return len(t.results)
}

// HaveRelativeStats is true for this object
func (t Object) HaveRelativeStats() bool {
	return true
}
