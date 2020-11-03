// Package memory_usage manages collecting data from performance_schema which holds
// information about memory usage
package memory_usage

import (
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql" // keep golint happy

	"github.com/sjmudd/ps-top/baseobject"
	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/logger"
)

const (
	description = "Memory Usage (memory_summary_global_by_event_name)"
)

// MemoryUsage represents a table of rows
type MemoryUsage struct {
	baseobject.BaseObject      // embedded
	last                  Rows // last loaded values
	Results               Rows // results (maybe with subtraction)
	Totals                Row  // totals of results
	db                    *sql.DB
}

func NewMemoryUsage(ctx *context.Context, db *sql.DB) *MemoryUsage {
	logger.Println("NewMemoryUsage()")
	mu := &MemoryUsage{
		db: db,
	}
	mu.SetContext(ctx)

	return mu
}

// Collect data from the db, no merging needed
func (mu *MemoryUsage) Collect() {
	mu.last = collect(mu.db)
	mu.SetLastCollectTime(time.Now())

	mu.makeResults()
}

// SetFirstFromLast resets the statistics to current values
func (mu *MemoryUsage) SetFirstFromLast() {

	mu.makeResults()
}

// Headings returns the headings for a table
func (mu MemoryUsage) Headings() string {
	var r Row

	return r.headings()
}

// RowContent returns the rows we need for displaying
func (mu MemoryUsage) RowContent() []string {
	rows := make([]string, 0, len(mu.Results))

	for i := range mu.Results {
		rows = append(rows, mu.Results[i].content(mu.Totals))
	}

	return rows
}

// Rows() returns the rows we have which are interesting
func (mu MemoryUsage) Rows() []Row {
	rows := make([]Row, 0, len(mu.Results))

	for i := range mu.Results {
		rows = append(rows, mu.Results[i])
	}

	return rows
}

// Totals return the row of totals
func (mu MemoryUsage) totals() Row {
	return mu.Totals
}

// TotalRowContent returns all the totals
func (mu MemoryUsage) TotalRowContent() string {
	return mu.Totals.content(mu.Totals)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (mu MemoryUsage) EmptyRowContent() string {
	var empty Row
	return empty.content(empty)
}

// Description provides a description of the table
func (mu MemoryUsage) Description() string {
	return description
}

// Len returns the length of the result set
func (mu MemoryUsage) Len() int {
	return len(mu.Results)
}

func (mu MemoryUsage) HaveRelativeStats() bool {
	return true
}

func (mu *MemoryUsage) makeResults() {
	mu.Results = make(Rows, len(mu.last))
	copy(mu.Results, mu.last)
	mu.Results.sort()
	mu.Totals = mu.Results.totals()
}
