// Package mutexlatency provides library routines for ps-top
// for managing the events_waits_summary_global_by_event_name table.
package mutexlatency

import (
	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model"
	"github.com/sjmudd/ps-top/model/common"
)

// MutexLatency holds a table of rows
type MutexLatency struct {
	*model.BaseCollector[Row, Rows]
}

// NewMutexLatency creates a new MutexLatency instance.
func NewMutexLatency(cfg *config.Config, db model.QueryExecutor) *MutexLatency {
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
	return &MutexLatency{BaseCollector: bc}
}

// Collect collects data from the db, updating first
// values if needed, and then subtracting first values if we want
// relative values, after which it stores totals.
func (ml *MutexLatency) Collect() {
	bc := ml.BaseCollector
	fetch := func() (Rows, error) {
		return collect(bc.DB()), nil
	}
	wantRefresh := func() bool {
		return (len(bc.First) == 0 && len(bc.Last) > 0) || totals(bc.First).SumTimerWait > totals(bc.Last).SumTimerWait
	}
	bc.Collect(fetch, wantRefresh)
}

// HaveRelativeStats is true for this object
func (ml MutexLatency) HaveRelativeStats() bool {
	return true
}

// WantRelativeStats returns the config setting.
func (ml MutexLatency) WantRelativeStats() bool {
	return ml.Config().WantRelativeStats()
}
