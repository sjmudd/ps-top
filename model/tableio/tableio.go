// Package tableio contains the routines for managing table_io_waits_by_table.
package tableio

import (
	"database/sql"
	"log"
	"time"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/common"
	"github.com/sjmudd/ps-top/utils"
)

// TableIo contains performance_schema.table_io_waits_summary_by_table data
type TableIo struct {
	config         *config.Config
	FirstCollected time.Time
	LastCollected  time.Time
	wantLatency    bool
	first          Rows // initial data for relative values
	last           Rows // last loaded values
	Results        Rows // results (maybe with subtraction)
	Totals         Row  // totals of results
	db             *sql.DB
}

// NewTableIo returns an i/o latency object with config and db handle
func NewTableIo(cfg *config.Config, db *sql.DB) *TableIo {
	tiol := &TableIo{
		config: cfg,
		db:     db,
	}

	return tiol
}

// ResetStatistics resets the statistics to current values
func (tiol *TableIo) ResetStatistics() {
	tiol.first = utils.DuplicateSlice(tiol.last)
	tiol.FirstCollected = tiol.LastCollected

	tiol.calculate()
}

// Collect collects data from the db, updating initial values
// if needed, and then subtracting initial values if we want relative
// values, after which it stores totals.
func (tiol *TableIo) Collect() {
	start := time.Now()

	tiol.last = collect(tiol.db, tiol.config.DatabaseFilter())
	tiol.LastCollected = time.Now()

	// check for no first data or need to reload initial characteristics
	if (len(tiol.first) == 0 && len(tiol.last) > 0) || totals(tiol.first).SumTimerWait > totals(tiol.last).SumTimerWait {
		tiol.first = utils.DuplicateSlice(tiol.last)
		tiol.FirstCollected = tiol.LastCollected
	}

	tiol.calculate()

	log.Println("tiol.first.totals():", totals(tiol.first))
	log.Println("tiol.last.totals():", totals(tiol.last))
	log.Println("TableIo.Collect() END, took:", time.Since(start))
}

func (tiol *TableIo) calculate() {
	tiol.Results = utils.DuplicateSlice(tiol.last)

	if tiol.config.WantRelativeStats() {
		common.SubtractByName(&tiol.Results, tiol.first,
			func(r Row) string { return r.Name },
			func(r *Row, o Row) { r.subtract(o) },
		)
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

// WantRelativeStats
func (tiol TableIo) WantRelativeStats() bool {
	return tiol.config.WantRelativeStats()
}
