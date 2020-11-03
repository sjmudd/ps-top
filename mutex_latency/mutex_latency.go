// Package mutex_latency provides library routines for ps-top.
// for managing the events_waits_summary_global_by_event_name table.
package mutex_latency

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/sjmudd/ps-top/baseobject"
	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/logger"
)

// MutexLatency holds a table of rows
type MutexLatency struct {
	baseobject.BaseObject      // embedded
	first                 Rows // initial data for relative values
	last                  Rows // last loaded values
	Results               Rows // results (maybe with subtraction)
	Totals                Row  // totals of results
	db                    *sql.DB
}

// NewMutexLatency returns a mutex latency object using given context and db
func NewMutexLatency(ctx *context.Context, db *sql.DB) *MutexLatency {
	logger.Println("NewMutexLatency()")
	if ctx == nil {
		log.Println("NewMutexLatency() ctx == nil!")
	}
	ml := &MutexLatency{
		db: db,
	}
	ml.SetContext(ctx)

	return ml
}

func (ml *MutexLatency) updateFirstFromLast() {
	ml.first = make(Rows, len(ml.last))
	ml.SetFirstCollectTime(ml.LastCollectTime())
	copy(ml.first, ml.last)
}

// Collect collects data from the db, updating first
// values if needed, and then subtracting first values if we want
// relative values, after which it stores totals.
func (ml *MutexLatency) Collect() {
	start := time.Now()
	// logger.Println("MutexLatency.Collect() BEGIN")
	ml.last = collect(ml.db)
	ml.SetLastCollectTime(time.Now())

	logger.Println("t.current collected", len(ml.last), "row(s) from SELECT")

	if len(ml.first) == 0 && len(ml.last) > 0 {
		logger.Println("ml.first: copying from ml.last (initial setup)")
		ml.updateFirstFromLast()
	}

	// check for reload initial characteristics
	if ml.first.needsRefresh(ml.last) {
		logger.Println("ml.first: copying from ml.last (data needs refreshing)")
		ml.updateFirstFromLast()
	}

	ml.makeResults()

	// logger.Println( "t.initial:", t.initial )
	// logger.Println( "t.current:", t.current )
	logger.Println("t.initial.totals():", ml.first.totals())
	logger.Println("t.current.totals():", ml.last.totals())
	// logger.Println("t.results:", ml.Results)
	// logger.Println("t.totals:", ml.Totals)
	logger.Println("MutexLatency.Collect() END, took:", time.Duration(time.Since(start)).String())
}

func (ml *MutexLatency) makeResults() {
	// logger.Println( "- t.results set from t.current" )
	ml.Results = make(Rows, len(ml.last))
	copy(ml.Results, ml.last)
	if ml.WantRelativeStats() {
		// logger.Println( "- subtracting t.initial from t.results as WantRelativeStats()" )
		ml.Results.subtract(ml.first)
	}

	// logger.Println( "- sorting t.results" )
	ml.Results.sort()
	// logger.Println( "- collecting t.totals from t.results" )
	ml.Totals = ml.Results.totals()
}

// SetFirstFromLast resets the statistics to current values
func (ml *MutexLatency) SetFirstFromLast() {
	ml.updateFirstFromLast()
	ml.makeResults()
}

// EmptyRowContent returns a string representation of no data
func (ml MutexLatency) EmptyRowContent() string {
	return ml.emptyRowContent()
}

// Headings returns a string representation of the headings
func (ml *MutexLatency) Headings() string {
	var r Row
	return r.headings()
}

// RowContent returns a string representation of the row content
func (ml MutexLatency) RowContent() []string {
	rows := make([]string, 0, len(ml.Results))

	for i := range ml.Results {
		rows = append(rows, ml.Results[i].content(ml.Totals))
	}

	return rows
}

func (ml MutexLatency) emptyRowContent() string {
	var r Row

	return r.content(r)
}

// TotalRowContent returns a string representation of the totals of the table
func (ml MutexLatency) TotalRowContent() string {
	return ml.Totals.content(ml.Totals)
}

// Description returns a description of the table
func (ml MutexLatency) Description() string {
	var count int
	for row := range ml.Results {
		if ml.Results[row].sumTimerWait > 0 {
			count++
		}
	}
	return fmt.Sprintf("Mutex Latency (events_waits_summary_global_by_event_name) %d rows", count)
}

// Len returns the length of the result set
func (ml MutexLatency) Len() int {
	return len(ml.Results)
}

// HaveRelativeStats is true for this object
func (ml MutexLatency) HaveRelativeStats() bool {
	return true
}
