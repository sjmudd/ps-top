// Package tablelocks represents the performance_schema table of the same name
package tablelocks

import (
	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model"
)

// TableLocks represents a table of rows
type TableLocks struct {
	*model.BaseCollector[Row, Rows]
}

// NewTableLocks creates a new TableLocks instance.
func NewTableLocks(cfg *config.Config, db model.QueryExecutor) *TableLocks {
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
	return &TableLocks{BaseCollector: bc}
}

// Collect data from the db, then merge it in.
func (tl *TableLocks) Collect() {
	fetch := func() (Rows, error) {
		return collect(tl.BaseCollector.DB(), tl.BaseCollector.Config().DatabaseFilter()), nil
	}
	wantRefresh := func() bool {
		bc := tl.BaseCollector
		return (len(bc.First) == 0 && len(bc.Last) > 0) || bc.First.needsRefresh(bc.Last)
	}
	tl.BaseCollector.Collect(fetch, wantRefresh)
}

// HaveRelativeStats is true for this object
func (tl TableLocks) HaveRelativeStats() bool {
	return true
}

// WantRelativeStats returns the config setting.
func (tl TableLocks) WantRelativeStats() bool {
	return tl.BaseCollector.Config().WantRelativeStats()
}
