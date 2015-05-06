// p_s - library routines for pstop.
//
// This file contains the library routines for managing the
// file_summary_by_instance table.
package file_summary_by_instance

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
	p_s.CollectionTime
	initial          table_rows
	current          table_rows
	results          table_rows
	totals           table_row
	global_variables map[string]string
}

// reset the statistics to current values
func (t *Object) SetInitialFromCurrent() {
	t.SetCollected()
	t.initial = make(table_rows, len(t.current))
	copy(t.initial, t.current)

	t.results = make(table_rows, len(t.current))
	copy(t.results, t.current)

	if t.WantRelativeStats() {
		t.results.subtract(t.initial) // should be 0 if relative
	}

	t.results.sort()
	t.totals = t.results.totals()
}

// Collect data from the db, then merge it in.
func (t *Object) Collect(dbh *sql.DB) {
	start := time.Now()
	// UPDATE current from db handle
	t.current = merge_by_table_name(select_rows(dbh), t.global_variables)

	// copy in initial data if it was not there
	if len(t.initial) == 0 && len(t.current) > 0 {
		t.initial = make(table_rows, len(t.current))
		copy(t.initial, t.current)
	}

	// check for reload initial characteristics
	if t.initial.needs_refresh(t.current) {
		t.initial = make(table_rows, len(t.current))
		copy(t.initial, t.current)
	}

	// update results to current value
	t.results = make(table_rows, len(t.current))
	copy(t.results, t.current)

	// make relative if need be
	if t.WantRelativeStats() {
		t.results.subtract(t.initial)
	}

	// sort the results
	t.results.sort()

	// setup the totals
	t.totals = t.results.totals()
	lib.Logger.Println("Object.Collect() took:", time.Duration(time.Since(start)).String())
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

// return the length of the result set
func (t Object) Len() int {
	return len(t.results)
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
	var count int
	for row := range t.results {
		if t.results[row].SUM_TIMER_WAIT > 0 {
			count++
		}
	}

	return fmt.Sprintf("File I/O Latency (file_summary_by_instance) %4d row(s)    ", count)
}

// create a new structure and include various variable values:
// - datadir, relay_log
// There's no checking that these are actually provided!
func NewFileSummaryByInstance(global_variables map[string]string) *Object {
	n := new(Object)

	n.global_variables = global_variables

	return n
}
