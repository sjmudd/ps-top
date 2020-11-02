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
	initial               Rows // initial data for relative values
	current               Rows // last loaded values
	results               Rows // results (maybe with subtraction)
	totals                Row  // totals of results
	db                    *sql.DB
}

func NewMutexLatency(ctx *context.Context, db *sql.DB) *MutexLatency {
	logger.Println("NewMutexLatency()")
	if ctx == nil {
		log.Println("NewMutexLatency() ctx == nil!")
	}
	o := &MutexLatency{
		db: db,
	}
	o.SetContext(ctx)

	return o
}

func (ml *MutexLatency) copyCurrentToInitial() {
	ml.initial = make(Rows, len(ml.current))
	ml.SetFirstCollectTime(ml.LastCollectTime())
	copy(ml.initial, ml.current)
}

// Collect collects data from the db, updating initial
// values if needed, and then subtracting initial values if we want
// relative values, after which it stores totals.
func (ml *MutexLatency) Collect() {
	start := time.Now()
	// logger.Println("MutexLatency.Collect() BEGIN")
	ml.current = selectRows(ml.db)
	ml.SetLastCollectTime(time.Now())

	logger.Println("t.current collected", len(ml.current), "row(s) from SELECT")

	if len(ml.initial) == 0 && len(ml.current) > 0 {
		logger.Println("ml.initial: copying from ml.current (initial setup)")
		ml.copyCurrentToInitial()
	}

	// check for reload initial characteristics
	if ml.initial.needsRefresh(ml.current) {
		logger.Println("ml.initial: copying from ml.current (data needs refreshing)")
		ml.copyCurrentToInitial()
	}

	ml.makeResults()

	// logger.Println( "t.initial:", t.initial )
	// logger.Println( "t.current:", t.current )
	logger.Println("t.initial.totals():", ml.initial.totals())
	logger.Println("t.current.totals():", ml.current.totals())
	// logger.Println("t.results:", ml.results)
	// logger.Println("t.totals:", ml.totals)
	logger.Println("MutexLatency.Collect() END, took:", time.Duration(time.Since(start)).String())
}

func (ml *MutexLatency) makeResults() {
	// logger.Println( "- t.results set from t.current" )
	ml.results = make(Rows, len(ml.current))
	copy(ml.results, ml.current)
	if ml.WantRelativeStats() {
		// logger.Println( "- subtracting t.initial from t.results as WantRelativeStats()" )
		ml.results.subtract(ml.initial)
	}

	// logger.Println( "- sorting t.results" )
	ml.results.sort()
	// logger.Println( "- collecting t.totals from t.results" )
	ml.totals = ml.results.totals()
}

// SetInitialFromCurrent resets the statistics to current values
func (ml *MutexLatency) SetInitialFromCurrent() {
	// logger.Println( "MutexLatency.SetInitialFromCurrent() BEGIN" )

	ml.copyCurrentToInitial()

	ml.makeResults()

	// logger.Println( "MutexLatency.SetInitialFromCurrent() END" )
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
	rows := make([]string, 0, len(ml.results))

	for i := range ml.results {
		rows = append(rows, ml.results[i].content(ml.totals))
	}

	return rows
}

func (ml MutexLatency) emptyRowContent() string {
	var r Row

	return r.content(r)
}

// TotalRowContent returns a string representation of the totals of the table
func (ml MutexLatency) TotalRowContent() string {
	return ml.totals.content(ml.totals)
}

// Description returns a description of the table
func (ml MutexLatency) Description() string {
	var count int
	for row := range ml.results {
		if ml.results[row].sumTimerWait > 0 {
			count++
		}
	}
	return fmt.Sprintf("Mutex Latency (events_waits_summary_global_by_event_name) %d rows", count)
}

// Len returns the length of the result set
func (ml MutexLatency) Len() int {
	return len(ml.results)
}

// HaveRelativeStats is true for this object
func (ml MutexLatency) HaveRelativeStats() bool {
	return true
}
