// Package tableio contains the routines for managing table_io_waits_by_table.
package tableio

import (
	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model"
	"github.com/sjmudd/ps-top/model/common"
)

// TableIo contains performance_schema.table_io_waits_summary_by_table data
type TableIo struct {
	*model.BaseCollector[Row, Rows]
	wantLatency bool
}

// NewTableIo creates a new TableIo instance.
func NewTableIo(cfg *config.Config, db model.QueryExecutor) *TableIo {
	process := func(last, first Rows) (Rows, Row) {
		results := make(Rows, len(last))
		copy(results, last)
		if cfg.WantRelativeStats() {
			common.SubtractByName(&results, first,
				func(r Row) string { return r.Name },
				func(r *Row, o Row) { r.subtract(o) },
			)
		}
		tot := totals(results)
		return results, tot
	}
	bc := model.NewBaseCollector[Row, Rows](cfg, db, process)
	return &TableIo{BaseCollector: bc, wantLatency: false}
}

// Collect collects data from the db, updating initial values
// if needed, and then subtracting initial values if we want relative
// values, after which it stores totals.
func (tiol *TableIo) Collect() {
	fetch := func() (Rows, error) {
		return collect(tiol.BaseCollector.DB(), tiol.BaseCollector.Config().DatabaseFilter()), nil
	}
	wantRefresh := func() bool {
		bc := tiol.BaseCollector
		return (len(bc.First) == 0 && len(bc.Last) > 0) || totals(bc.First).SumTimerWait > totals(bc.Last).SumTimerWait
	}
	tiol.BaseCollector.Collect(fetch, wantRefresh)
}

// WantsLatency returns whether we want to see latency information
func (tiol *TableIo) WantsLatency() bool {
	return tiol.wantLatency
}

// HaveRelativeStats is true for this object
func (tiol TableIo) HaveRelativeStats() bool {
	return true
}

// WantRelativeStats returns whether relative stats are desired based on config
func (tiol TableIo) WantRelativeStats() bool {
	return tiol.BaseCollector.Config().WantRelativeStats()
}
