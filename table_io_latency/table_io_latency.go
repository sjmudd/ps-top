// Package table_io_latency contains the routines for managing table_io_waits_by_table.
package table_io_latency

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/sjmudd/ps-top/baseobject"
	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/logger"
)

// TableIoLatency contains performance_schema.table_io_waits_summary_by_table data
type TableIoLatency struct {
	baseobject.BaseObject
	wantLatency bool
	initial     Rows   // initial data for relative values
	current     Rows   // last loaded values
	results     Rows   // results (maybe with subtraction)
	totals      Row    // totals of results
	descStart   string // start of description
	db          *sql.DB
}

func NewTableIoLatency(ctx *context.Context, db *sql.DB) *TableIoLatency {
	if ctx == nil {
		log.Fatal("NewTableIoLatency() ctx should not be nil")
	}
	tiol := &TableIoLatency{
		db: db,
	}
	tiol.SetContext(ctx)

	return tiol
}

func (tiol *TableIoLatency) copyCurrentToInitial() {
	tiol.initial = make(Rows, len(tiol.current))
	tiol.SetFirstCollectTime(tiol.LastCollectTime())
	copy(tiol.initial, tiol.current)
}

// Collect collects data from the db, updating initial values
// if needed, and then subtracting initial values if we want relative
// values, after which it stores totals.
func (tiol *TableIoLatency) Collect() {
	start := time.Now()
	// logger.Println("TableIoLatency.Collect() BEGIN")
	tiol.current = selectRows(tiol.db)
	tiol.SetLastCollectTime(time.Now())
	logger.Println("t.current collected", len(tiol.current), "row(s) from SELECT")

	if len(tiol.initial) == 0 && len(tiol.current) > 0 {
		logger.Println("tiol.initial: copying from tiol.current (initial setup)")
		tiol.copyCurrentToInitial()
	}

	// check for reload initial characteristics
	if tiol.initial.needsRefresh(tiol.current) {
		logger.Println("tiol.initial: copying from t.current (data needs refreshing)")
		tiol.copyCurrentToInitial()
	}

	tiol.makeResults()

	// logger.Println( "t.initial:", t.initial )
	// logger.Println( "t.current:", t.current )
	logger.Println("tiol.initial.totals():", tiol.initial.totals())
	logger.Println("tiol.current.totals():", tiol.current.totals())
	// logger.Println("tiol.results:", tiol.results)
	// logger.Println("tiol.totals:", tiol.totals)
	logger.Println("TableIoLatency.Collect() END, took:", time.Duration(time.Since(start)).String())
}

func (tiol *TableIoLatency) makeResults() {
	logger.Println("table_io_latency.makeResults()")
	logger.Println("- HaveRelativeStats()", tiol.HaveRelativeStats())
	logger.Println("- WantRelativeStats()", tiol.WantRelativeStats())
	tiol.results = make(Rows, len(tiol.current))
	copy(tiol.results, tiol.current)
	if tiol.WantRelativeStats() {
		logger.Println("- subtracting t.initial from t.results as WantRelativeStats()")
		tiol.results.subtract(tiol.initial)
	}

	// logger.Println( "- sorting t.results" )
	tiol.results.sort(tiol.wantLatency)
	// logger.Println( "- collecting t.totals from t.results" )
	tiol.totals = tiol.results.totals()
}

// SetInitialFromCurrent resets the statistics to current values
func (tiol *TableIoLatency) SetInitialFromCurrent() {
	// logger.Println( "TableIoLatency.SetInitialFromCurrent() BEGIN" )

	tiol.copyCurrentToInitial()

	tiol.makeResults()

	// logger.Println( "TableIoLatency.SetInitialFromCurrent() END" )
}

// Headings returns the headings for the table
func (tiol TableIoLatency) Headings() string {
	var r Row

	if tiol.wantLatency {
		return r.latencyHeadings()
	}

	return r.opsHeadings()
}

// RowContent returns the top maxRows data from the table
func (tiol TableIoLatency) RowContent() []string {
	rows := make([]string, 0, len(tiol.results))

	for i := range tiol.results {
		if tiol.wantLatency {
			rows = append(rows, tiol.results[i].latencyRowContent(tiol.totals))
		} else {
			rows = append(rows, tiol.results[i].opsRowContent(tiol.totals))
		}
	}

	return rows
}

// EmptyRowContent returns an empty row
func (tiol TableIoLatency) EmptyRowContent() string {
	var r Row

	if tiol.wantLatency {
		return r.latencyRowContent(r)
	}

	return r.opsRowContent(r)
}

// TotalRowContent returns a formated row containing totals data
func (tiol TableIoLatency) TotalRowContent() string {
	if tiol.wantLatency {
		return tiol.totals.latencyRowContent(tiol.totals)
	}

	return tiol.totals.opsRowContent(tiol.totals)
}

// Description returns the description of the table as a string
func (tiol TableIoLatency) Description() string {
	var count int
	for row := range tiol.results {
		if tiol.results[row].sumTimerWait > 0 {
			count++
		}
	}

	return fmt.Sprintf("Table %s (table_io_waits_summary_by_table) %d rows", tiol.descStart, count)
}

// Len returns the length of the result set
func (tiol TableIoLatency) Len() int {
	return len(tiol.current)
}

// SetWantsLatency allows us to define if we want latency settings
func (tiol *TableIoLatency) SetWantsLatency(wantLatency bool) {
	tiol.wantLatency = wantLatency
}

// WantsLatency returns whether we want to see latency information
func (tiol TableIoLatency) WantsLatency() bool {
	return tiol.wantLatency
}

// HaveRelativeStats is true for this object
func (tiol TableIoLatency) HaveRelativeStats() bool {
	return true
}
