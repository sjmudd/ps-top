// Package mutexlatency provides library routines for ps-top.
// for managing the events_waits_summary_global_by_event_name table.
package mutexlatency

import (
	"database/sql"
	"log"
	"time"

	"github.com/sjmudd/ps-top/baseobject"
	"github.com/sjmudd/ps-top/context"
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
	log.Println("NewMutexLatency()")
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
	// log.Println("MutexLatency.Collect() BEGIN")
	ml.last = collect(ml.db)
	ml.SetLastCollectTime(time.Now())

	log.Println("t.current collected", len(ml.last), "row(s) from SELECT")

	if len(ml.first) == 0 && len(ml.last) > 0 {
		log.Println("ml.first: copying from ml.last (initial setup)")
		ml.updateFirstFromLast()
	}

	// check for reload initial characteristics
	if ml.first.needsRefresh(ml.last) {
		log.Println("ml.first: copying from ml.last (data needs refreshing)")
		ml.updateFirstFromLast()
	}

	ml.makeResults()

	// log.Println( "t.initial:", t.initial )
	// log.Println( "t.current:", t.current )
	log.Println("t.initial.totals():", ml.first.totals())
	log.Println("t.current.totals():", ml.last.totals())
	// log.Println("t.results:", ml.Results)
	// log.Println("t.totals:", ml.Totals)
	log.Println("MutexLatency.Collect() END, took:", time.Duration(time.Since(start)).String())
}

func (ml *MutexLatency) makeResults() {
	// log.Println( "- t.results set from t.current" )
	ml.Results = make(Rows, len(ml.last))
	copy(ml.Results, ml.last)
	if ml.WantRelativeStats() {
		// log.Println( "- subtracting t.initial from t.results as WantRelativeStats()" )
		ml.Results.subtract(ml.first)
	}

	ml.Totals = ml.Results.totals()
}

// SetFirstFromLast resets the statistics to current values
func (ml *MutexLatency) SetFirstFromLast() {
	ml.updateFirstFromLast()
	ml.makeResults()
}

// HaveRelativeStats is true for this object
func (ml MutexLatency) HaveRelativeStats() bool {
	return true
}
