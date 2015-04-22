// p_s - library routines for pstop.
//
// This file contains the library routines for managing
// performance_schema.table_io_waits_by_table.
package table_io_waits_summary_by_table

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
	desc_start   string     // start of description
}

func (t *Object) SetWantsLatency(want_latency bool) {
	t.want_latency = want_latency
	if t.want_latency {
		t.desc_start = "Latency"
	} else {
		t.desc_start = "Operations"
	}
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
	t.results.Sort(t.want_latency)
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

func (o *Object) Headings() string {
	var r table_row

	if o.want_latency {
		return r.latency_headings()
	} else {
		return r.ops_headings()
	}
}

func (t Object) RowContent(max_rows int) []string {
	rows := make([]string, 0, max_rows)

	for i := range t.results {
		if i < max_rows {
			if t.want_latency {
				rows = append(rows, t.results[i].latency_row_content(t.totals))
			} else {
				rows = append(rows, t.results[i].ops_row_content(t.totals))
			}
		}
	}

	return rows
}

func (t Object) EmptyRowContent() string {
	var r table_row

	if t.want_latency {
		return r.latency_row_content(r)
	} else {
		return r.ops_row_content(r)
	}
}

func (t Object) TotalRowContent() string {
	if t.want_latency {
		return t.totals.latency_row_content(t.totals)
	} else {
		return t.totals.ops_row_content(t.totals)
	}
}

func (t Object) Description() string {
	var count int
	for row := range t.results {
		if t.results[row].SUM_TIMER_WAIT > 0 {
			count++
		}
	}

	return fmt.Sprintf("%s by Table Name (table_io_waits_summary_by_table) %d rows", t.desc_start, count )
}
