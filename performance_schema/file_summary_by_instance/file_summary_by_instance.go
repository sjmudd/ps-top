// performance_schema - library routines for pstop.
//
// This file contains the library routines for managing the
// file_summary_by_instance table.
package file_summary_by_instance

import (
	"database/sql"
	"time"

	"github.com/sjmudd/pstop/lib"
	ps "github.com/sjmudd/pstop/performance_schema"
)

/*
CREATE TABLE `file_summary_by_instance` (
  `FILE_NAME` varchar(512) NOT NULL,
  `EVENT_NAME` varchar(128) NOT NULL,				// not collected
  `OBJECT_INSTANCE_BEGIN` bigint(20) unsigned NOT NULL,		// not collected
  `COUNT_STAR` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WAIT` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WAIT` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WAIT` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WAIT` bigint(20) unsigned NOT NULL,
  `COUNT_READ` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_READ` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_READ` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_READ` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_READ` bigint(20) unsigned NOT NULL,
  `SUM_NUMBER_OF_BYTES_READ` bigint(20) NOT NULL,
  `COUNT_WRITE` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `SUM_NUMBER_OF_BYTES_WRITE` bigint(20) NOT NULL,
  `COUNT_MISC` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_MISC` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_MISC` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_MISC` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_MISC` bigint(20) unsigned NOT NULL
) ENGINE=PERFORMANCE_SCHEMA DEFAULT CHARSET=utf8
1 row in set (0.00 sec)

*/

// a table of rows
type File_summary_by_instance struct {
	ps.RelativeStats
	ps.InitialTime
	initial          file_summary_by_instance_rows
	current          file_summary_by_instance_rows
	results          file_summary_by_instance_rows
	totals           file_summary_by_instance_row
	global_variables map[string]string
}

// reset the statistics to current values
func (t *File_summary_by_instance) SyncReferenceValues() {
	t.SetNow()
	t.initial = make(file_summary_by_instance_rows, len(t.current))
	copy(t.initial, t.current)

	t.results = make(file_summary_by_instance_rows, len(t.current))
	copy(t.results, t.current)

	if t.WantRelativeStats() {
		t.results.subtract(t.initial) // should be 0 if relative
	}

	t.results.sort()
	t.totals = t.results.totals()
}

// Collect data from the db, then merge it in.
func (t *File_summary_by_instance) Collect(dbh *sql.DB) {
	start := time.Now()
	// UPDATE current from db handle
	t.current = merge_by_table_name(select_fsbi_rows(dbh), t.global_variables)

	// copy in initial data if it was not there
	if len(t.initial) == 0 && len(t.current) > 0 {
		t.initial = make(file_summary_by_instance_rows, len(t.current))
		copy(t.initial, t.current)
	}

	// check for reload initial characteristics
	if t.initial.needs_refresh(t.current) {
		t.initial = make(file_summary_by_instance_rows, len(t.current))
		copy(t.initial, t.current)
	}

	// update results to current value
	t.results = make(file_summary_by_instance_rows, len(t.current))
	copy(t.results, t.current)

	// make relative if need be
	if t.WantRelativeStats() {
		t.results.subtract(t.initial)
	}

	// sort the results
	t.results.sort()

	// setup the totals
	t.totals = t.results.totals()
	lib.Logger.Println("File_summary_by_instance.Collect() took:", time.Duration(time.Since(start)).String())
}

// return the headings for a table
func (t File_summary_by_instance) Headings() string {
	var r file_summary_by_instance_row

	return r.headings()
}

// return the rows we need for displaying
func (t File_summary_by_instance) RowContent(max_rows int) []string {
	rows := make([]string, 0, max_rows)

	for i := range t.results {
		if i < max_rows {
			rows = append(rows, t.results[i].row_content(t.totals))
		}
	}

	return rows
}

// return all the totals
func (t File_summary_by_instance) TotalRowContent() string {
	return t.totals.row_content(t.totals)
}

// return an empty string of data (for filling in)
func (t File_summary_by_instance) EmptyRowContent() string {
	var emtpy file_summary_by_instance_row
	return emtpy.row_content(emtpy)
}

func (t File_summary_by_instance) Description() string {
	return "File I/O by filename (file_summary_by_instance)"
}

// create a new structure and include various variable values:
// - datadir, relay_log
// There's no checking that these are actually provided!
func NewFileSummaryByInstance(global_variables map[string]string) *File_summary_by_instance {
	n := new(File_summary_by_instance)

	n.global_variables = global_variables

	return n
}
