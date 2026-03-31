// Package stageslatency is the nterface to events_stages_summary_global_by_event_name
package stageslatency

import (
	"github.com/sjmudd/ps-top/model"
	"github.com/sjmudd/ps-top/model/common"
)

// StagesLatency provides a public view of object
type StagesLatency struct {
	*model.BaseCollector[Row, Rows]
}

// NewStagesLatency creates a new StagesLatency instance.
func NewStagesLatency(cfg model.Config, db model.QueryExecutor) *StagesLatency {
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
	return &StagesLatency{BaseCollector: bc}
}

// Collect collects data from the db, updating initial
// values if needed, and then subtracting initial values if we want
// relative values, after which it stores totals.
func (sl *StagesLatency) Collect() {
	bc := sl.BaseCollector
	fetch := func() (Rows, error) {
		return collect(bc.DB()), nil
	}
	wantRefresh := func() bool {
		return (len(bc.First) == 0 && len(bc.Last) > 0) || totals(bc.First).SumTimerWait > totals(bc.Last).SumTimerWait
	}
	bc.Collect(fetch, wantRefresh)
}

// HaveRelativeStats is true for this object
func (sl StagesLatency) HaveRelativeStats() bool {
	return true
}

// WantRelativeStats returns the config setting.
func (sl StagesLatency) WantRelativeStats() bool {
	return sl.Config().WantRelativeStats()
}
