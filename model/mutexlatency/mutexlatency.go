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

// Collect collects data from the db, updating first
// values if needed, and then subtracting first values if we want
// relative values, after which it stores totals.
func (ml *MutexLatency) Collect() {
	start := time.Now()

	ml.last = collect(ml.db)
	ml.LastCollected = time.Now()

	// check if no first data or we need to reload initial characteristics
	if (len(ml.first) == 0 && len(ml.last) > 0) || ml.first.needsRefresh(ml.last) {
		ml.first = duplicateSlice(ml.last)
		ml.FirstCollected = ml.LastCollected
	}

	ml.calculate()

	log.Println("t.initial.totals():", totals(ml.first))
	log.Println("t.current.totals():", totals(ml.last))
	log.Println("MutexLatency.Collect() END, took:", time.Duration(time.Since(start)).String())
}

func (ml *MutexLatency) calculate() {
	// log.Println( "- t.results set from t.current" )
	ml.Results = make(Rows, len(ml.last))
	copy(ml.Results, ml.last)
	if ml.WantRelativeStats() {
		// log.Println( "- subtracting t.initial from t.results as WantRelativeStats()" )
		ml.Results.subtract(ml.first)
	}

	ml.Totals = totals(ml.Results)
}

// ResetStatistics resets the statistics to current values
func (ml *MutexLatency) ResetStatistics() {
	ml.first = duplicateSlice(ml.last)
	ml.FirstCollected = ml.LastCollected

	ml.calculate()
}

// HaveRelativeStats is true for this object
func (ml MutexLatency) HaveRelativeStats() bool {
	return true
}
