// Package table_io_latency contains the routines for managing table_io_waits_by_table.
package table_io_latency

import (
	"database/sql"
	"time"

	"github.com/sjmudd/ps-top/baseobject"
	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/logger"
)

// TableIoLatency contains performance_schema.table_io_waits_summary_by_table data
type TableIoLatency struct {
	baseobject.BaseObject
	wantLatency bool
	first       Rows // initial data for relative values
	last        Rows // last loaded values
	Results     Rows // results (maybe with subtraction)
	Totals      Row  // totals of results
	db          *sql.DB
}

// NewTableIoLatency returns an i/o latency object with context and db handle
func NewTableIoLatency(ctx *context.Context, db *sql.DB) *TableIoLatency {
	tiol := &TableIoLatency{
		db: db,
	}
	tiol.SetContext(ctx)

	return tiol
}

// SetFirstFromLast resets the statistics to current values
func (tiol *TableIoLatency) SetFirstFromLast() {
	tiol.updateFirstFromLast()
	tiol.makeResults()
}

func (tiol *TableIoLatency) updateFirstFromLast() {
	tiol.first = make([]Row, len(tiol.last))
	copy(tiol.first, tiol.last)
	tiol.SetFirstCollectTime(tiol.LastCollectTime())
}

// Collect collects data from the db, updating initial values
// if needed, and then subtracting initial values if we want relative
// values, after which it stores totals.
func (tiol *TableIoLatency) Collect() {
	start := time.Now()
	// logger.Println("TableIoLatency.Collect() BEGIN")
	tiol.last = collect(tiol.db)
	tiol.SetLastCollectTime(time.Now())
	logger.Println("t.current collected", len(tiol.last), "row(s) from SELECT")

	if len(tiol.first) == 0 && len(tiol.last) > 0 {
		logger.Println("tiol.first: copying from tiol.last (initial setup)")
		tiol.updateFirstFromLast()
	}

	// check for reload initial characteristics
	if tiol.first.needsRefresh(tiol.last) {
		logger.Println("tiol.first: copying from t.current (data needs refreshing)")
		tiol.updateFirstFromLast()
	}

	tiol.makeResults()

	logger.Println("tiol.first.totals():", tiol.first.totals())
	logger.Println("tiol.last.totals():", tiol.last.totals())
	logger.Println("TableIoLatency.Collect() END, took:", time.Duration(time.Since(start)).String())
}

func (tiol *TableIoLatency) makeResults() {
	tiol.Results = make([]Row, len(tiol.last))
	copy(tiol.Results, tiol.last)
	if tiol.WantRelativeStats() {
		tiol.Results.subtract(tiol.first)
	}

	tiol.Results.sort(tiol.wantLatency)
	tiol.Totals = tiol.Results.totals()
}

// Len returns the length of the result set
func (tiol TableIoLatency) Len() int {
	return len(tiol.last)
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
