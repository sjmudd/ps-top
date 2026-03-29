// Package memoryusage manages collecting data from performance_schema which holds
// information about memory usage
package memoryusage

import (
	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model"
)

// MemoryUsage represents a table of rows
type MemoryUsage struct {
	*model.BaseCollector[Row, []Row]
}

// NewMemoryUsage returns a pointer to a MemoryUsage struct
func NewMemoryUsage(cfg *config.Config, db model.QueryExecutor) *MemoryUsage {
	process := func(last, _ []Row) ([]Row, Row) {
		results := make([]Row, len(last))
		copy(results, last)
		tot := totals(results)
		return results, tot
	}
	bc := model.NewBaseCollector[Row, []Row](cfg, db, process)
	return &MemoryUsage{BaseCollector: bc}
}

// Collect data from the db, no merging needed
func (mu *MemoryUsage) Collect() {
	bc := mu.BaseCollector
	fetch := func() ([]Row, error) {
		return collect(bc.DB()), nil
	}
	wantRefresh := func() bool {
		// MemoryUsage does not support relative stats, so always refresh baseline to current
		return true
	}
	bc.Collect(fetch, wantRefresh)
}

// Rows returns the rows we have which are interesting
func (mu *MemoryUsage) Rows() []Row {
	return mu.Results
}

// HaveRelativeStats returns if the values returned are relative to a previous collection
func (mu *MemoryUsage) HaveRelativeStats() bool {
	return false
}

// WantRelativeStats returns whether relative stats are desired based on config
func (mu *MemoryUsage) WantRelativeStats() bool {
	return mu.Config().WantRelativeStats()
}
