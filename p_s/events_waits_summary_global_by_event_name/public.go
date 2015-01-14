// p_s - library routines for pstop.
//
// This file contains the library routines for managing the
// events_waits_summary_global_by_event_name table.
package events_waits_summary_global_by_event_name

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sjmudd/pstop/lib"
	"github.com/sjmudd/pstop/p_s"
)

// a table of rows
type Object struct {
	p_s.RelativeStats
	p_s.InitialTime
	want_latency bool
	initial      table_rows // initial data for relative values
	current      table_rows // last loaded values
	results      table_rows // results (maybe with subtraction)
	totals       table_row  // totals of results
}

func (t *Object) SetWantsLatency(want_latency bool) {
	t.want_latency = want_latency
}

func (t Object) WantsLatency() bool {
	return t.want_latency
}

// Collect() collects data from the db, updating initial
// values if needed, and then subtracting initial values if we want
// relative values, after which it stores totals.
func (t *Object) Collect(dbh *sql.DB) {
	start := time.Now()
	// lib.Logger.Println("Object.Collect() BEGIN")
	t.current = select_rows(dbh)
	lib.Logger.Println("t.current collected", len(t.current), "row(s) from SELECT")

	if len(t.initial) == 0 && len(t.current) > 0 {
		lib.Logger.Println("t.initial: copying from t.current (initial setup)")
		t.initial = make(table_rows, len(t.current))
		copy(t.initial, t.current)
	}

	// check for reload initial characteristics
	if t.initial.needs_refresh(t.current) {
		lib.Logger.Println("t.initial: copying from t.current (data needs refreshing)")
		t.initial = make(table_rows, len(t.current))
		copy(t.initial, t.current)
	}

	t.make_results()

	// lib.Logger.Println( "t.initial:", t.initial )
	// lib.Logger.Println( "t.current:", t.current )
	lib.Logger.Println("t.initial.totals():", t.initial.totals())
	lib.Logger.Println("t.current.totals():", t.current.totals())
	// lib.Logger.Println("t.results:", t.results)
	// lib.Logger.Println("t.totals:", t.totals)
	lib.Logger.Println("Object.Collect() END, took:", time.Duration(time.Since(start)).String())
}

func (t *Object) make_results() {
	// lib.Logger.Println( "- t.results set from t.current" )
	t.results = make(table_rows, len(t.current))
	copy(t.results, t.current)
	if t.WantRelativeStats() {
		// lib.Logger.Println( "- subtracting t.initial from t.results as WantRelativeStats()" )
		t.results.subtract(t.initial)
	}

	// lib.Logger.Println( "- sorting t.results" )
	t.results.Sort()
	// lib.Logger.Println( "- collecting t.totals from t.results" )
	t.totals = t.results.totals()
}

// reset the statistics to current values
func (t *Object) SyncReferenceValues() {
	// lib.Logger.Println( "Object.SyncReferenceValues() BEGIN" )

	t.SetNow()
	t.initial = make(table_rows, len(t.current))
	copy(t.initial, t.current)

	t.make_results()

	// lib.Logger.Println( "Object.SyncReferenceValues() END" )
}

func (t Object) EmptyRowContent() string {
	return t.emptyRowContent()
}

func (t *Object) Headings() string {
	var r table_row

	return r.headings()
}

func (t Object) RowContent(max_rows int) []string {
	rows := make([]string, 0, max_rows)

	for i := range t.results {
		if i < max_rows {
			rows = append(rows, t.results[i].row_content(t.totals))
		}
	}

	return rows
}

func (t Object) emptyRowContent() string {
	var r table_row

	return r.row_content(r)
}

func (t Object) TotalRowContent() string {
	return t.totals.row_content(t.totals)
}

func (t Object) Description() string {
	count := t.count_rows()
	return fmt.Sprintf("Mutex Latency (events_waits_summary_global_by_event_name) %d rows", count)
}

func (t Object) count_rows() int {
	var count int
	for row := range t.results {
		if t.results[row].SUM_TIMER_WAIT > 0 {
			count++
		}
	}
	return count
}