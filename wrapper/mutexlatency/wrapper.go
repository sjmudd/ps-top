// Package mutexlatency holds the routines which manage the server mutexes.
package mutexlatency

import (
	"database/sql"
	"fmt"
	"slices"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/mutexlatency"
	"github.com/sjmudd/ps-top/utils"
	"github.com/sjmudd/ps-top/wrapper"
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

// Wrapper wraps a MutexLatency struct.
type Wrapper struct {
	*wrapper.BaseWrapper[mutexlatency.Row, *mutexlatency.MutexLatency]
}

// NewMutexLatency creates a wrapper around mutexlatency.
func NewMutexLatency(cfg *config.Config, db *sql.DB) *Wrapper {
	ml := mutexlatency.NewMutexLatency(cfg, db)
	bw := wrapper.NewBaseWrapper(
		ml,
		"Mutex Latency (events_waits_summary_global_by_event_name)",
		defaultSort,
		defaultHasData,
		defaultContent,
	)
	return &Wrapper{BaseWrapper: bw}
}

// Headings returns the headings for a table.
func (w *Wrapper) Headings() string {
	return fmt.Sprintf("%10s %8s %8s|%s", "Latency", "MtxCnt", "%", "Mutex Name")
}
