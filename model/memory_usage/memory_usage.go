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

// Len returns the length of the result set
func (mu MemoryUsage) Len() int {
	return len(mu.Results)
}

func (mu MemoryUsage) HaveRelativeStats() bool {
	return false
}

func (mu *MemoryUsage) makeResults() {
	mu.Results = make(Rows, len(mu.last))
	copy(mu.Results, mu.last)
	mu.Results.sort()
	mu.Totals = mu.Results.totals()
}
