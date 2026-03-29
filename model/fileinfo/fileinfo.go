// Package fileinfo holds the routines which manage the file_summary_by_instance table.
package fileinfo

import (
	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model"
)

// FileIoLatency represents the contents of the data collected from file_summary_by_instance
type FileIoLatency struct {
	*model.BaseCollector[Row, Rows]
}

// NewFileSummaryByInstance creates a new structure and include various variable values:
// - datadir, relay_log
// There's no checking that these are actually provided!
func NewFileSummaryByInstance(cfg *config.Config, db model.QueryExecutor) *FileIoLatency {
	process := func(last, first Rows) (Rows, Row) {
		results := make(Rows, len(last))
		copy(results, last)
		if cfg.WantRelativeStats() {
			results.subtract(first)
		}
		tot := totals(results)
		return results, tot
	}
	bc := model.NewBaseCollector[Row, Rows](cfg, db, process)
	return &FileIoLatency{BaseCollector: bc}
}

// Collect data from the db, then merge it in.
func (fiol *FileIoLatency) Collect() {
	bc := fiol.BaseCollector
	fetch := func() (Rows, error) {
		raw := collect(bc.DB())
		// Apply transformation using config variables
		transformed := FileInfo2MySQLNames(
			bc.Config().Variables().Get("datadir"),
			bc.Config().Variables().Get("relaylog"),
			raw,
		)
		return transformed, nil
	}
	wantRefresh := func() bool {
		return (len(bc.First) == 0 && len(bc.Last) > 0) || totals(bc.First).SumTimerWait > totals(bc.Last).SumTimerWait
	}
	bc.Collect(fetch, wantRefresh)
}

// HaveRelativeStats is true for this object
func (fiol FileIoLatency) HaveRelativeStats() bool {
	return true
}

// WantRelativeStats returns the config setting.
func (fiol FileIoLatency) WantRelativeStats() bool {
	return fiol.Config().WantRelativeStats()
}
