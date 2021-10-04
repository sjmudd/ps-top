// Package fileinfo holds the routines which manage the file_summary_by_instance table.
package fileinfo

import (
	"database/sql"
	"log"
	"time"

	"github.com/sjmudd/ps-top/baseobject"
	"github.com/sjmudd/ps-top/context"
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

// ResetStatistics resets the statistics to last values
func (fiol *FileIoLatency) ResetStatistics() {
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

	log.Println("fiol.first.totals():", fiol.first.totals())
	log.Println("fiol.last.totals():", fiol.last.totals())
	log.Println("FileIoLatency.Collect() took:", time.Duration(time.Since(start)).String())
}

func (fiol *FileIoLatency) makeResults() {
	fiol.Results = make([]Row, len(fiol.last))
	copy(fiol.Results, fiol.last)
	if fiol.WantRelativeStats() {
		fiol.Results.subtract(fiol.first)
	}

	fiol.Totals = fiol.Results.totals()
}

// Len return the length of the result set
func (fiol FileIoLatency) Len() int {
	return len(fiol.Results)
}

// HaveRelativeStats is true for this object
func (fiol FileIoLatency) HaveRelativeStats() bool {
	return true
}
