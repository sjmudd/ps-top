// Package stageslatency holds the routines which manage the stages table.
package stageslatency

import (
	"database/sql"
	"fmt"
	"slices"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/stageslatency"
	"github.com/sjmudd/ps-top/utils"
	"github.com/sjmudd/ps-top/wrapper"
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

// Wrapper wraps a Stages struct.
type Wrapper struct {
	*wrapper.BaseWrapper[stageslatency.Row, *stageslatency.StagesLatency]
}

// NewStagesLatency creates a wrapper around stageslatency.
func NewStagesLatency(cfg *config.Config, db *sql.DB) *Wrapper {
	sl := stageslatency.NewStagesLatency(cfg, db)
	bw := wrapper.NewBaseWrapper(
		sl,
		"SQL Stage Latency (events_stages_summary_global_by_event_name)",
		defaultSort,
		defaultHasData,
		defaultContent,
	)
	return &Wrapper{BaseWrapper: bw}
}

// Headings returns the headings for a table.
func (w *Wrapper) Headings() string {
	return fmt.Sprintf("%10s %6s %8s|%s", "Latency", "%", "Counter", "Stage Name")
}
