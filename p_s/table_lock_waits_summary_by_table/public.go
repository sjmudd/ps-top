// This file contains the library routines for managing the
// table_lock_waits_summary_by_table table.
package table_lock_waits_summary_by_table

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"time"

	"github.com/sjmudd/pstop/lib"
	"github.com/sjmudd/pstop/p_s"
)

// a table of rows
type Object struct {
	p_s.RelativeStats
	p_s.InitialTime
	initial table_rows // initial data for relative values
	current table_rows // last loaded values
	results table_rows // results (maybe with subtraction)
	totals  table_row  // totals of results
}

// Collect data from the db, then merge it in.
func (t *Object) Collect(dbh *sql.DB) {
	start := time.Now()
	t.current = select_rows(dbh)

	if len(t.initial) == 0 && len(t.current) > 0 {
		t.initial = make(table_rows, len(t.current))
		copy(t.initial, t.current)
	}

	// check for reload initial characteristics
	if t.initial.needs_refresh(t.current) {
		t.initial = make(table_rows, len(t.current))
		copy(t.initial, t.current)
	}

	t.make_results()
	lib.Logger.Println("Object.Collect() took:", time.Duration(time.Since(start)).String())
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
	t.results.sort()
	// lib.Logger.Println( "- collecting t.totals from t.results" )
	t.totals = t.results.totals()
}

// reset the statistics to current values
func (t *Object) SyncReferenceValues() {
	t.SetNow()
	t.initial = make(table_rows, len(t.current))
	copy(t.initial, t.current)

	t.make_results()
}

// return the headings for a table
func (t Object) Headings() string {
	var r table_row

	return r.headings()
}

// return the rows we need for displaying
func (t Object) RowContent(max_rows int) []string {
	rows := make([]string, 0, max_rows)

	for i := range t.results {
		if i < max_rows {
			rows = append(rows, t.results[i].row_content(t.totals))
		}
	}

	return rows
}

// return all the totals
func (t Object) TotalRowContent() string {
	return t.totals.row_content(t.totals)
}

// return an empty string of data (for filling in)
func (t Object) EmptyRowContent() string {
	var emtpy table_row
	return emtpy.row_content(emtpy)
}

func (t Object) Description() string {
	return "Locks by Table Name (table_lock_waits_summary_by_table)"
}
