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
	first                 Rows
	last                  Rows
	Results               Rows
	Totals                Row
	db                    *sql.DB
}

// NewFileSummaryByInstance creates a new structure and include various variable values:
// - datadir, relay_log
// There's no checking that these are actually provided!
func NewFileSummaryByInstance(ctx *context.Context, db *sql.DB) *FileIoLatency {
	fiol := &FileIoLatency{
		db: db,
	}
	fiol.SetContext(ctx)

	return fiol
}

// SetFirstFromLast resets the statistics to last values
func (fiol *FileIoLatency) SetFirstFromLast() {
	fiol.updateFirstFromLast()
	fiol.makeResults()
}

func (fiol *FileIoLatency) updateFirstFromLast() {
	fiol.first = make([]Row, len(fiol.last))
	copy(fiol.first, fiol.last)
	fiol.SetFirstCollectTime(fiol.LastCollectTime())
}

// Collect data from the db, then merge it in.
func (fiol *FileIoLatency) Collect() {
	start := time.Now()
	fiol.last = collect(fiol.db).mergeByName(fiol.Variables())
	fiol.SetLastCollectTime(time.Now())

	// copy in first data if it was not there
	if len(fiol.first) == 0 && len(fiol.last) > 0 {
		fiol.updateFirstFromLast()
	}

	// check for reload initial characteristics
	if fiol.first.needsRefresh(fiol.last) {
		fiol.updateFirstFromLast()
	}

	fiol.makeResults()

	logger.Println("fiol.first.totals():", fiol.first.totals())
	logger.Println("fiol.last.totals():", fiol.last.totals())
	logger.Println("FileIoLatency.Collect() took:", time.Duration(time.Since(start)).String())
}

func (fiol *FileIoLatency) makeResults() {
	fiol.Results = make([]Row, len(fiol.last))
	copy(fiol.Results, fiol.last)
	if fiol.WantRelativeStats() {
		fiol.Results.subtract(fiol.first)
	}

	fiol.Results.sort()
	fiol.Totals = fiol.Results.totals()
}

// Headings returns the headings for a table
func (fiol FileIoLatency) Headings() string {
	var r Row

	return r.headings()
}

// RowContent returns the rows we need for displaying
func (fiol FileIoLatency) RowContent() []string {
	rows := make([]string, 0, len(fiol.Results))

	for i := range fiol.Results {
		rows = append(rows, fiol.Results[i].content(fiol.Totals))
	}

	return rows
}

// Len return the length of the result set
func (fiol FileIoLatency) Len() int {
	return len(fiol.Results)
}

// TotalRowContent returns all the totals
func (fiol FileIoLatency) TotalRowContent() string {
	return fiol.Totals.content(fiol.Totals)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (fiol FileIoLatency) EmptyRowContent() string {
	var empty Row
	return empty.content(empty)
}

// Description returns a description of the table
func (fiol FileIoLatency) Description() string {
	var count int
	for row := range fiol.Results {
		if fiol.Results[row].sumTimerWait > 0 {
			count++
		}
	}

	return fmt.Sprintf("File I/O Latency (file_summary_by_instance) %4d row(s)    ", count)
}

// HaveRelativeStats is true for this object
func (fiol FileIoLatency) HaveRelativeStats() bool {
	return true
}
