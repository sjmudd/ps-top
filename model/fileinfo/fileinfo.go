// Package fileinfo holds the routines which manage the file_summary_by_instance table.
package fileinfo

import (
	"database/sql"
	"log"
	"time"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/utils"
)

// FileIoLatency represents the contents of the data collected from file_summary_by_instance
type FileIoLatency struct {
	config         *config.Config
	FirstCollected time.Time // the first collection time (for relative data)
	LastCollected  time.Time // the last collection time
	first          Rows
	last           Rows
	Results        Rows
	Totals         Row
	db             *sql.DB
}

// NewFileSummaryByInstance creates a new structure and include various variable values:
// - datadir, relay_log
// There's no checking that these are actually provided!
func NewFileSummaryByInstance(cfg *config.Config, db *sql.DB) *FileIoLatency {
	fiol := &FileIoLatency{
		db:     db,
		config: cfg,
	}

	return fiol
}

// ResetStatistics resets the statistics to last values
func (fiol *FileIoLatency) ResetStatistics() {
	fiol.first = utils.DuplicateSlice(fiol.last)
	fiol.FirstCollected = fiol.LastCollected

	fiol.calculate()
}

// Collect data from the db, then merge it in.
func (fiol *FileIoLatency) Collect() {
	start := time.Now()
	fiol.last = FileInfo2MySQLNames(fiol.config.Variables(), collect(fiol.db))
	fiol.LastCollected = time.Now()

	// copy in first data if it was not there
	// or check for reload initial characteristics
	if (len(fiol.first) == 0 && len(fiol.last) > 0) || fiol.first.needsRefresh(fiol.last) {
		fiol.first = utils.DuplicateSlice(fiol.last)
		fiol.FirstCollected = fiol.LastCollected
	}

	fiol.calculate()

	log.Println("fiol.first.totals():", totals(fiol.first))
	log.Println("fiol.last.totals():", totals(fiol.last))
	log.Println("FileIoLatency.Collect() took:", time.Duration(time.Since(start)).String())
}

func (fiol *FileIoLatency) calculate() {
	fiol.Results = utils.DuplicateSlice(fiol.last)

	if fiol.config.WantRelativeStats() {
		fiol.Results.subtract(fiol.first)
	}

	fiol.Totals = totals(fiol.Results)
}

// HaveRelativeStats is true for this object
func (fiol FileIoLatency) HaveRelativeStats() bool {
	return true
}

// WantRelativeStats
func (fiol FileIoLatency) WantRelativeStats() bool {
	return fiol.config.WantRelativeStats()
}
