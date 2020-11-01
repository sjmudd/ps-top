// Package memory_usage manages collecting data from performance_schema which holds
// information about memory usage
package memory_usage

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql" // keep golint happy

	"github.com/sjmudd/ps-top/baseobject"
	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/logger"
)

const (
	description = "Memory Usage (memory_summary_global_by_event_name)"
)

// Object represents a table of rows
type Object struct {
	baseobject.BaseObject      // embedded
	current               Rows // last loaded values
	results               Rows // results (maybe with subtraction)
	totals                Row  // totals of results
	db                    *sql.DB
}

func NewMemoryUsage(ctx *context.Context, db *sql.DB) *Object {
	logger.Println("NewMemoryUsage()")
	o := &Object{
		db: db,
	}
	o.SetContext(ctx)

	return o
}

// Collect data from the db, no merging needed
func (t *Object) Collect() {
	t.current = selectRows(t.db)
	t.SetLastCollectTimeNow()

	t.makeResults()
}

// SetInitialFromCurrent resets the statistics to current values
func (t *Object) SetInitialFromCurrent() {

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

// Rows() returns the rows we have which are interesting
func (t Object) Rows() []Row {
	rows := make([]Row, 0, len(t.results))

	for i := range t.results {
		rows = append(rows, t.results[i])
	}

	return rows
}

// Totals return the row of totals
func (t Object) Totals() Row {
	return t.totals
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

func (t Object) HaveRelativeStats() bool {
	return true
}
