// Package table_io contains the routines for managing table_io_waits_by_table.
package table_io

import (
	"database/sql"
	"time"

	"github.com/sjmudd/ps-top/baseobject"
	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/logger"
)

// TableIo contains performance_schema.table_io_waits_summary_by_table data
type TableIo struct {
	baseobject.BaseObject
	wantLatency bool
	first       Rows // initial data for relative values
	last        Rows // last loaded values
	Results     Rows // results (maybe with subtraction)
	Totals      Row  // totals of results
	db          *sql.DB
}

// NewTableIo returns an i/o latency object with context and db handle
func NewTableIo(ctx *context.Context, db *sql.DB) *TableIo {
	tiol := &TableIo{
		db: db,
	}
	tiol.SetContext(ctx)

	return tiol
}

// SetFirstFromLast resets the statistics to current values
func (tiol *TableIo) SetFirstFromLast() {
	tiol.updateFirstFromLast()
	tiol.makeResults()
}

func (tiol *TableIo) updateFirstFromLast() {
	tiol.first = make([]Row, len(tiol.last))
	copy(tiol.first, tiol.last)
	tiol.SetFirstCollectTime(tiol.LastCollectTime())
}

// Collect collects data from the db, updating initial values
// if needed, and then subtracting initial values if we want relative
// values, after which it stores totals.
func (tiol *TableIo) Collect() {
	start := time.Now()
	// logger.Println("TableIo.Collect() BEGIN")
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
	logger.Println("TableIo.Collect() END, took:", time.Duration(time.Since(start)).String())
}

func (tiol *TableIo) makeResults() {
	tiol.Results = make([]Row, len(tiol.last))
	copy(tiol.Results, tiol.last)
	if tiol.WantRelativeStats() {
		tiol.Results.subtract(tiol.first)
	}

	tiol.Results.sort(tiol.wantLatency)
	tiol.Totals = tiol.Results.totals()
}

// Len returns the length of the result set
func (tiol TableIo) Len() int {
	return len(tiol.last)
}

// SetWantsLatency allows us to define if we want latency settings
func (tiol *TableIo) SetWantsLatency(wantLatency bool) {
	tiol.wantLatency = wantLatency
}

// WantsLatency returns whether we want to see latency information
func (tiol TableIo) WantsLatency() bool {
	return tiol.wantLatency
}

// HaveRelativeStats is true for this object
func (tiol TableIo) HaveRelativeStats() bool {
	return true
}
