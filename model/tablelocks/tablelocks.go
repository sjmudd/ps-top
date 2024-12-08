// Package tablelocks represents the performance_schema table of the same name
package tablelocks

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql" // keep golint happy
	"log"
	"time"

	"github.com/sjmudd/ps-top/baseobject"
	"github.com/sjmudd/ps-top/config"
)

// TableLocks represents a table of rows
type TableLocks struct {
	baseobject.BaseObject
	initial Rows // initial data for relative values
	current Rows // last loaded values
	Results Rows // results (maybe with subtraction)
	Totals  Row  // totals of results
	db      *sql.DB
}

// NewTableLocks returns a pointer to an object of this type
func NewTableLocks(cfg *config.Config, db *sql.DB) *TableLocks {
	tl := &TableLocks{
		db: db,
	}
	tl.SetConfig(cfg)

	return tl
}

func (tl *TableLocks) copyCurrentToInitial() {
	tl.initial = make(Rows, len(tl.current))
	copy(tl.initial, tl.current)
	tl.FirstCollected = tl.LastCollected
}

// Collect data from the db, then merge it in.
func (tl *TableLocks) Collect() {
	start := time.Now()
	tl.current = collect(tl.db, tl.DatabaseFilter())
	tl.LastCollected = time.Now()

	// check for no data or check for reload initial characteristics
	if (len(tl.initial) == 0 && len(tl.current) > 0) || tl.initial.needsRefresh(tl.current) {
		tl.copyCurrentToInitial()
	}

	tl.calculate()
	log.Println("TableLocks.Collect() took:", time.Duration(time.Since(start)).String())
}

func (tl *TableLocks) calculate() {
	tl.Results = make(Rows, len(tl.current))
	copy(tl.Results, tl.current)
	if tl.WantRelativeStats() {
		tl.Results.subtract(tl.initial)
	}
	tl.Totals = totals(tl.Results)
}

// ResetStatistics resets the statistics to current values
func (tl *TableLocks) ResetStatistics() {
	tl.copyCurrentToInitial()
	tl.calculate()
}

// HaveRelativeStats is true for this object
func (tl TableLocks) HaveRelativeStats() bool {
	return true
}
