// Package tableio contains the routines for managing table_io_waits_by_table.
package tableio

import (
	"database/sql"
	"log"
	"time"

	"github.com/sjmudd/ps-top/baseobject"
	"github.com/sjmudd/ps-top/context"
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

// ResetStatistics resets the statistics to current values
func (tiol *TableIo) ResetStatistics() {
	tiol.first = duplicateSlice(tiol.last)
	tiol.FirstCollected = tiol.LastCollected

	tiol.calculate()
}

// Collect collects data from the db, updating initial values
// if needed, and then subtracting initial values if we want relative
// values, after which it stores totals.
func (tiol *TableIo) Collect() {
	start := time.Now()

	tiol.last = collect(tiol.db, tiol.DatabaseFilter())
	tiol.LastCollected = time.Now()

	// check for no first data or need to reload initial characteristics
	if (len(tiol.first) == 0 && len(tiol.last) > 0) || tiol.first.needsRefresh(tiol.last) {
		tiol.first = duplicateSlice(tiol.last)
		tiol.FirstCollected = tiol.LastCollected
	}

	tiol.calculate()

	log.Println("tiol.first.totals():", totals(tiol.first))
	log.Println("tiol.last.totals():", totals(tiol.last))
	log.Println("TableIo.Collect() END, took:", time.Duration(time.Since(start)).String())
}

func (tiol *TableIo) calculate() {
	tiol.Results = duplicateSlice(tiol.last)

	if tiol.WantRelativeStats() {
		tiol.Results.subtract(tiol.first)
	}

	tiol.Totals = totals(tiol.Results)
}

// WantsLatency returns whether we want to see latency information
func (tiol TableIo) WantsLatency() bool {
	return tiol.wantLatency
}

// HaveRelativeStats is true for this object
func (tiol TableIo) HaveRelativeStats() bool {
	return true
}
