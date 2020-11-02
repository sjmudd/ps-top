// Package file_io_latency holds the routines which manage the file_summary_by_instance table.
package file_io_latency

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sjmudd/ps-top/baseobject"
	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/logger"
)

// FileIoLatency represents the contents of the data collected from file_summary_by_instance
type FileIoLatency struct {
	baseobject.BaseObject // embedded
	initial               Rows
	current               Rows
	results               Rows
	totals                Row
	db                    *sql.DB
}

// NewFileSummaryByInstance creates a new structure and include various variable values:
// - datadir, relay_log
// There's no checking that these are actually provided!
func NewFileSummaryByInstance(ctx *context.Context, db *sql.DB) *FileIoLatency {
	logger.Println("NewFileSummaryByInstance()")
	fiol := &FileIoLatency{
		db: db,
	}
	fiol.SetContext(ctx)

	return fiol
}

// SetInitialFromCurrent resets the statistics to current values
func (fiol *FileIoLatency) SetInitialFromCurrent() {
	fiol.copyCurrentToInitial()

	fiol.makeResults()
}

func (fiol *FileIoLatency) copyCurrentToInitial() {
	fiol.initial = make(Rows, len(fiol.current))
	fiol.SetFirstCollectTime(fiol.LastCollectTime())
	copy(fiol.initial, fiol.current)
}

// Collect data from the db, then merge it in.
func (fiol *FileIoLatency) Collect() {
	fiol.current = selectRows(fiol.db).mergeByName(fiol.Variables())
	fiol.SetLastCollectTime(time.Now())

	// copy in initial data if it was not there
	if len(fiol.initial) == 0 && len(fiol.current) > 0 {
		fiol.copyCurrentToInitial()
	}

	// check for reload initial characteristics
	if fiol.initial.needsRefresh(fiol.current) {
		fiol.copyCurrentToInitial()
	}

	fiol.makeResults()
}

func (fiol *FileIoLatency) makeResults() {
	fiol.results = make(Rows, len(fiol.current))
	copy(fiol.results, fiol.current)
	if fiol.WantRelativeStats() {
		fiol.results.subtract(fiol.initial)
	}

	fiol.results.sort()
	fiol.totals = fiol.results.totals()
}

// Headings returns the headings for a table
func (fiol FileIoLatency) Headings() string {
	var r Row

	return r.headings()
}

// RowContent returns the rows we need for displaying
func (fiol FileIoLatency) RowContent() []string {
	rows := make([]string, 0, len(fiol.results))

	for i := range fiol.results {
		rows = append(rows, fiol.results[i].content(fiol.totals))
	}

	return rows
}

// Len return the length of the result set
func (fiol FileIoLatency) Len() int {
	return len(fiol.results)
}

// TotalRowContent returns all the totals
func (fiol FileIoLatency) TotalRowContent() string {
	return fiol.totals.content(fiol.totals)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (fiol FileIoLatency) EmptyRowContent() string {
	var empty Row
	return empty.content(empty)
}

// Description returns a description of the table
func (fiol FileIoLatency) Description() string {
	var count int
	for row := range fiol.results {
		if fiol.results[row].sumTimerWait > 0 {
			count++
		}
	}

	return fmt.Sprintf("File I/O Latency (file_summary_by_instance) %4d row(s)    ", count)
}

// HaveRelativeStats is true for this object
func (fiol FileIoLatency) HaveRelativeStats() bool {
	return true
}
