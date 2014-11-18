// performance_schema - library routines for pstop.
//
// This file contains the library routines for managing the
// table_io_waits_by_table table.
package table_io_waits_summary_by_table

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sjmudd/pstop/lib"
	ps "github.com/sjmudd/pstop/performance_schema"
)

// a table of rows
type Table_io_waits_summary_by_table struct {
	ps.RelativeStats
	ps.InitialTime
	want_latency bool
	initial      table_io_waits_summary_by_table_rows // initial data for relative values
	current      table_io_waits_summary_by_table_rows // last loaded values
	results      table_io_waits_summary_by_table_rows // results (maybe with subtraction)
	totals       table_io_waits_summary_by_table_row  // totals of results
}

func (t *Table_io_waits_summary_by_table) SetWantsLatency(want_latency bool) {
	t.want_latency = want_latency
}

func (t Table_io_waits_summary_by_table) WantsLatency() bool {
	return t.want_latency
}

// Collect() collects data from the db, updating initial
// values if needed, and then subtracting initial values if we want
// relative values, after which it stores totals.
func (t *Table_io_waits_summary_by_table) Collect(dbh *sql.DB) {
	start := time.Now()
	// lib.Logger.Println("Table_io_waits_summary_by_table.Collect() BEGIN")
	t.current = select_tiwsbt_rows(dbh)
	lib.Logger.Println("t.current collected", len(t.current), "row(s) from SELECT")

	if len(t.initial) == 0 && len(t.current) > 0 {
		lib.Logger.Println("t.initial: copying from t.current (initial setup)" )
		t.initial = make(table_io_waits_summary_by_table_rows, len(t.current))
		copy(t.initial, t.current)
	}

	// check for reload initial characteristics
	if t.initial.needs_refresh(t.current) {
		lib.Logger.Println( "t.initial: copying from t.current (data needs refreshing)" )
		t.initial = make(table_io_waits_summary_by_table_rows, len(t.current))
		copy(t.initial, t.current)
	}

	t.make_results()

	// lib.Logger.Println( "t.initial:", t.initial )
	// lib.Logger.Println( "t.current:", t.current )
	lib.Logger.Println("t.initial.totals():", t.initial.totals() )
	lib.Logger.Println("t.current.totals():", t.current.totals() )
	// lib.Logger.Println("t.results:", t.results)
	// lib.Logger.Println("t.totals:", t.totals)
	lib.Logger.Println("Table_io_waits_summary_by_table.Collect() END, took:", time.Duration(time.Since(start)).String())
}

func (t *Table_io_waits_summary_by_table) make_results() {
	// lib.Logger.Println( "- t.results set from t.current" )
	t.results = make(table_io_waits_summary_by_table_rows, len(t.current))
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
func (t *Table_io_waits_summary_by_table) SyncReferenceValues() {
	// lib.Logger.Println( "Table_io_waits_summary_by_table.SyncReferenceValues() BEGIN" )

	t.initial = make(table_io_waits_summary_by_table_rows, len(t.current))
	copy(t.initial, t.current)

	t.make_results()

	// lib.Logger.Println( "Table_io_waits_summary_by_table.SyncReferenceValues() END" )
}

func (t *Table_io_waits_summary_by_table) Headings() string {
	if t.want_latency {
		return t.latencyHeadings()
	} else {
		return t.opsHeadings()
	}
}

func (t Table_io_waits_summary_by_table) RowContent(max_rows int) []string {
	if t.want_latency {
		return t.latencyRowContent(max_rows)
	} else {
		return t.opsRowContent(max_rows)
	}
}

func (t Table_io_waits_summary_by_table) EmptyRowContent() string {
	if t.want_latency {
		return t.emptyLatencyRowContent()
	} else {
		return t.emptyOpsRowContent()
	}
}

func (t Table_io_waits_summary_by_table) TotalRowContent() string {
	if t.want_latency {
		return t.totalLatencyRowContent()
	} else {
		return t.totalOpsRowContent()
	}
}

func (t Table_io_waits_summary_by_table) Description() string {
	if t.want_latency {
		return t.latencyDescription()
	} else {
		return t.opsDescription()
	}
}

func (t *Table_io_waits_summary_by_table) latencyHeadings() string {
	var r table_io_waits_summary_by_table_row

	return r.latency_headings()
}

func (t *Table_io_waits_summary_by_table) opsHeadings() string {
	var r table_io_waits_summary_by_table_row

	return r.ops_headings()
}

func (t Table_io_waits_summary_by_table) opsRowContent(max_rows int) []string {
	rows := make([]string, 0, max_rows)

	for i := range t.results {
		if i < max_rows {
			rows = append(rows, t.results[i].ops_row_content(t.totals))
		}
	}

	return rows
}

func (t Table_io_waits_summary_by_table) latencyRowContent(max_rows int) []string {
	rows := make([]string, 0, max_rows)

	for i := range t.results {
		if i < max_rows {
			rows = append(rows, t.results[i].latency_row_content(t.totals))
		}
	}

	return rows
}

func (t Table_io_waits_summary_by_table) emptyOpsRowContent() string {
	var r table_io_waits_summary_by_table_row

	return r.ops_row_content(r)
}

func (t Table_io_waits_summary_by_table) emptyLatencyRowContent() string {
	var r table_io_waits_summary_by_table_row

	return r.latency_row_content(r)
}

func (t Table_io_waits_summary_by_table) totalOpsRowContent() string {
	return t.totals.ops_row_content(t.totals)
}

func (t Table_io_waits_summary_by_table) totalLatencyRowContent() string {
	return t.totals.latency_row_content(t.totals)
}

func (t Table_io_waits_summary_by_table) latencyDescription() string {
	count := t.count_rows()
	return fmt.Sprintf("Latency by Table Name (table_io_waits_summary_by_table) %d rows", count)
}

func (t Table_io_waits_summary_by_table) opsDescription() string {
	count := t.count_rows()
	return fmt.Sprintf("Operations by Table Name (table_io_waits_summary_by_table) %d rows", count)
}

func (t Table_io_waits_summary_by_table) count_rows() int {
	var count int
	for row := range t.results {
		if t.results[row].SUM_TIMER_WAIT > 0 {
			count++
		}
	}
	return count
}
