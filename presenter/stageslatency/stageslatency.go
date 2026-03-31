// Package stageslatency holds the routines which manage the stages table.
package stageslatency

import (
	"database/sql"
	"fmt"
	"slices"

	"github.com/sjmudd/ps-top/model"
	"github.com/sjmudd/ps-top/model/stageslatency"
	"github.com/sjmudd/ps-top/presenter"
	"github.com/sjmudd/ps-top/utils"
)

var (
	defaultSort = func(rows []stageslatency.Row) {
		slices.SortFunc(rows, func(a, b stageslatency.Row) int {
			return utils.SumTimerWaitNameOrdering(
				utils.NewSumTimerWaitName(a.Name, a.SumTimerWait),
				utils.NewSumTimerWaitName(b.Name, b.SumTimerWait),
			)
		})
	}

	defaultHasData = func(r stageslatency.Row) bool { return r.SumTimerWait > 0 }

	defaultContent = func(row, totals stageslatency.Row) string {
		name := row.Name
		if row.CountStar == 0 && name != "Totals" {
			name = ""
		}
		return fmt.Sprintf("%10s %6s %8s|%s",
			utils.FormatTime(row.SumTimerWait),
			utils.FormatPct(utils.Divide(row.SumTimerWait, totals.SumTimerWait)),
			utils.FormatAmount(row.CountStar),
			name)
	}
)

// Presenter presents a StagesLatency struct.
type Presenter struct {
	*presenter.BasePresenter[stageslatency.Row, *stageslatency.StagesLatency]
}

// NewStagesLatency creates a presenter for stageslatency.
func NewStagesLatency(cfg model.Config, db *sql.DB) *Presenter {
	sl := stageslatency.NewStagesLatency(cfg, db)
	bp := presenter.NewBasePresenter(
		sl,
		"SQL Stage Latency (events_stages_summary_global_by_event_name)",
		defaultSort,
		defaultHasData,
		defaultContent,
	)
	return &Presenter{BasePresenter: bp}
}

// Headings returns the headings for a table.
func (p *Presenter) Headings() string {
	return fmt.Sprintf("%10s %6s %8s|%s", "Latency", "%", "Counter", "Stage Name")
}
