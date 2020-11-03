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
	Results Rows // results (maybe with subtraction)
	Totals  Row  // totals of results
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
	tll.current = collect(tll.db)
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
	tll.Results = make(Rows, len(tll.current))
	copy(tll.Results, tll.current)
	if tll.WantRelativeStats() {
		tll.Results.subtract(tll.initial)
	}

	tll.Results.sort()
	tll.Totals = tll.Results.totals()
}

// SetFirstFromLast resets the statistics to current values
func (tll *TableLockLatency) SetFirstFromLast() {
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
	rows := make([]string, 0, len(tll.Results))

	for i := range tll.Results {
		rows = append(rows, tll.Results[i].content(tll.Totals))
	}

	return rows
}

// TotalRowContent returns all the totals
func (tll TableLockLatency) TotalRowContent() string {
	return tll.Totals.content(tll.Totals)
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
	return len(tll.Results)
}

// HaveRelativeStats is true for this object
func (tll TableLockLatency) HaveRelativeStats() bool {
	return true
}
