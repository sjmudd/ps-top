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

// Object represents a table of rows
type Object struct {
	baseobject.BaseObject
	initial Rows // initial data for relative values
	current Rows // last loaded values
	results Rows // results (maybe with subtraction)
	totals  Row  // totals of results
	db      *sql.DB
}

// NewTableLockLatency returns a pointer to an object of this type
func NewTableLockLatency(ctx *context.Context, db *sql.DB) *Object {
	o := &Object{
		db: db,
	}
	o.SetContext(ctx)

	return o
}

func (t *Object) copyCurrentToInitial() {
	t.initial = make(Rows, len(t.current))
	t.SetInitialCollectTime(t.LastCollectTime())
	copy(t.initial, t.current)
}

// Collect data from the db, then merge it in.
func (t *Object) Collect() {
	start := time.Now()
	t.current = selectRows(t.db)
	t.SetLastCollectTimeNow()

	if len(t.initial) == 0 && len(t.current) > 0 {
		t.copyCurrentToInitial()
	}

	// check for reload initial characteristics
	if t.initial.needsRefresh(t.current) {
		t.copyCurrentToInitial()
	}

	t.makeResults()
	logger.Println("Object.Collect() took:", time.Duration(time.Since(start)).String())
}

func (t *Object) makeResults() {
	t.results = make(Rows, len(t.current))
	copy(t.results, t.current)
	if t.WantRelativeStats() {
		t.results.subtract(t.initial)
	}

	t.results.sort()
	t.totals = t.results.totals()
}

// SetInitialFromCurrent resets the statistics to current values
func (t *Object) SetInitialFromCurrent() {
	t.copyCurrentToInitial()
	t.makeResults()
}

// Headings returns the headings for a table
func (t Object) Headings() string {
	var r Row

	return r.headings()
}

// RowContent returns the rows we need for displaying
func (t Object) RowContent() []string {
	rows := make([]string, 0, len(t.results))

	for i := range t.results {
		rows = append(rows, t.results[i].rowContent(t.totals))
	}

	return rows
}

// TotalRowContent returns all the totals
func (t Object) TotalRowContent() string {
	return t.totals.rowContent(t.totals)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (t Object) EmptyRowContent() string {
	var empty Row
	return empty.rowContent(empty)
}

// Description provides a description of the table
func (t Object) Description() string {
	return description
}

// Len returns the length of the result set
func (t Object) Len() int {
	return len(t.results)
}

// HaveRelativeStats is true for this object
func (t Object) HaveRelativeStats() bool {
	return true
}
