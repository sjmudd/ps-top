// Package mutexlatency holds the routines which manage the server mutexes.
package mutexlatency

import (
	"database/sql"
	"fmt"
	"slices"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/mutexlatency"
	"github.com/sjmudd/ps-top/presenter"
	"github.com/sjmudd/ps-top/utils"
)

var (
	defaultSort = func(rows []mutexlatency.Row) {
		slices.SortFunc(rows, func(a, b mutexlatency.Row) int {
			return utils.SumTimerWaitNameOrdering(
				utils.NewSumTimerWaitName(a.Name, a.SumTimerWait),
				utils.NewSumTimerWaitName(b.Name, b.SumTimerWait),
			)
		})
	}

	defaultHasData = func(r mutexlatency.Row) bool { return r.SumTimerWait > 0 }

	defaultContent = func(row, totals mutexlatency.Row) string {
		name := row.Name
		if row.CountStar == 0 && name != "Totals" {
			name = ""
		}
		return fmt.Sprintf("%10s %8s %8s|%s",
			utils.FormatTime(row.SumTimerWait),
			utils.FormatAmount(row.CountStar),
			utils.FormatPct(utils.Divide(row.SumTimerWait, totals.SumTimerWait)),
			name)
	}
)

// Presenter presents a MutexLatency struct.
type Presenter struct {
	*presenter.BasePresenter[mutexlatency.Row, *mutexlatency.MutexLatency]
}

// NewMutexLatency creates a presenter for mutexlatency.
func NewMutexLatency(cfg *config.Config, db *sql.DB) *Presenter {
	ml := mutexlatency.NewMutexLatency(cfg, db)
	bp := presenter.NewBasePresenter(
		ml,
		"Mutex Latency (events_waits_summary_global_by_event_name)",
		defaultSort,
		defaultHasData,
		defaultContent,
	)
	return &Presenter{BasePresenter: bp}
}

// Headings returns the headings for a table.
func (p *Presenter) Headings() string {
	return fmt.Sprintf("%10s %8s %8s|%s", "Latency", "MtxCnt", "%", "Mutex Name")
}
