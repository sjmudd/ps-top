// Package table_lock_latency represents the performance_schema table of the same name
package table_lock_latency

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql" // keep golint happy
	"time"

	"github.com/sjmudd/ps-top/baseobject"
	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/logger"
)

const (
	description = "Locks by Table Name (table_lock_waits_summary_by_table)"
)

// TableLockLatency represents a table of rows
type TableLockLatency struct {
	baseobject.BaseObject
	initial Rows // initial data for relative values
	current Rows // last loaded values
	results Rows // results (maybe with subtraction)
	totals  Row  // totals of results
	db      *sql.DB
}

// NewTableLockLatency returns a pointer to an object of this type
func NewTableLockLatency(ctx *context.Context, db *sql.DB) *TableLockLatency {
	tll := &TableLockLatency{
		db: db,
	}
	tll.SetContext(ctx)

	return tll
}

func (tll *TableLockLatency) copyCurrentToInitial() {
	tll.initial = make(Rows, len(tll.current))
	tll.SetFirstCollectTime(tll.LastCollectTime())
	copy(tll.initial, tll.current)
}

// Collect data from the db, then merge it in.
func (tll *TableLockLatency) Collect() {
	start := time.Now()
	tll.current = selectRows(tll.db)
	tll.SetLastCollectTime(time.Now())

	if len(tll.initial) == 0 && len(tll.current) > 0 {
		tll.copyCurrentToInitial()
	}

	// check for reload initial characteristics
	if tll.initial.needsRefresh(tll.current) {
		tll.copyCurrentToInitial()
	}

	tll.makeResults()
	logger.Println("TableLockLatency.Collect() took:", time.Duration(time.Since(start)).String())
}

func (tll *TableLockLatency) makeResults() {
	tll.results = make(Rows, len(tll.current))
	copy(tll.results, tll.current)
	if tll.WantRelativeStats() {
		tll.results.subtract(tll.initial)
	}

	tll.results.sort()
	tll.totals = tll.results.totals()
}

// SetInitialFromCurrent resets the statistics to current values
func (tll *TableLockLatency) SetInitialFromCurrent() {
	tll.copyCurrentToInitial()
	tll.makeResults()
}

// Headings returns the headings for a table
func (tll TableLockLatency) Headings() string {
	var r Row

	return r.headings()
}

// RowContent returns the rows we need for displaying
func (tll TableLockLatency) RowContent() []string {
	rows := make([]string, 0, len(tll.results))

	for i := range tll.results {
		rows = append(rows, tll.results[i].content(tll.totals))
	}

	return rows
}

// TotalRowContent returns all the totals
func (tll TableLockLatency) TotalRowContent() string {
	return tll.totals.content(tll.totals)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (tll TableLockLatency) EmptyRowContent() string {
	var empty Row
	return empty.content(empty)
}

// Description provides a description of the table
func (tll TableLockLatency) Description() string {
	return description
}

// Len returns the length of the result set
func (tll TableLockLatency) Len() int {
	return len(tll.results)
}

// HaveRelativeStats is true for this object
func (tll TableLockLatency) HaveRelativeStats() bool {
	return true
}
