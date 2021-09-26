// Package table_locks represents the performance_schema table of the same name
package table_locks

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql" // keep golint happy
	"log"
	"time"

	"github.com/sjmudd/ps-top/baseobject"
	"github.com/sjmudd/ps-top/context"
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
func NewTableLocks(ctx *context.Context, db *sql.DB) *TableLocks {
	tll := &TableLocks{
		db: db,
	}
	tll.SetContext(ctx)

	return tll
}

func (tll *TableLocks) copyCurrentToInitial() {
	tll.initial = make(Rows, len(tll.current))
	tll.SetFirstCollectTime(tll.LastCollectTime())
	copy(tll.initial, tll.current)
}

// Collect data from the db, then merge it in.
func (tll *TableLocks) Collect() {
	start := time.Now()
	tll.current = collect(tll.db, tll.DatabaseFilter())
	tll.SetLastCollectTime(time.Now())

	if len(tll.initial) == 0 && len(tll.current) > 0 {
		tll.copyCurrentToInitial()
	}

	// check for reload initial characteristics
	if tll.initial.needsRefresh(tll.current) {
		tll.copyCurrentToInitial()
	}

	tll.makeResults()
	log.Println("TableLocks.Collect() took:", time.Duration(time.Since(start)).String())
}

func (tll *TableLocks) makeResults() {
	tll.Results = make(Rows, len(tll.current))
	copy(tll.Results, tll.current)
	if tll.WantRelativeStats() {
		tll.Results.subtract(tll.initial)
	}
	tll.Totals = tll.Results.totals()
}

// SetFirstFromLast resets the statistics to current values
func (tll *TableLocks) SetFirstFromLast() {
	tll.copyCurrentToInitial()
	tll.makeResults()
}

// HaveRelativeStats is true for this object
func (tll TableLocks) HaveRelativeStats() bool {
	return true
}
