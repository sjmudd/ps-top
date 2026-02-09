// Package stageslatency holds the routines which manage the stages table.
package stageslatency

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/stageslatency"
	"github.com/sjmudd/ps-top/utils"
	"github.com/sjmudd/ps-top/wrapper"
)

// Wrapper wraps a Stages struct
type Wrapper struct {
	sl *stageslatency.StagesLatency
}

// NewStagesLatency creates a wrapper around stageslatency
func NewStagesLatency(cfg *config.Config, db *sql.DB) *Wrapper {
	return &Wrapper{
		sl: stageslatency.NewStagesLatency(cfg, db),
	}
}

// ResetStatistics resets the statistics to last values
func (slw *Wrapper) ResetStatistics() {
	slw.sl.ResetStatistics()
}

// Collect data from the db, then merge it in.
func (slw *Wrapper) Collect() {
	slw.sl.Collect()
	sort.Slice(slw.sl.Results, func(i, j int) bool {
		return (slw.sl.Results[i].SumTimerWait > slw.sl.Results[j].SumTimerWait) ||
			((slw.sl.Results[i].SumTimerWait == slw.sl.Results[j].SumTimerWait) && (slw.sl.Results[i].Name < slw.sl.Results[j].Name))
	})
}

// Headings returns the headings for a table
func (slw Wrapper) Headings() string {
	return fmt.Sprintf("%10s %6s %8s|%s", "Latency", "%", "Counter", "Stage Name")

}

// RowContent returns the rows we need for displaying
func (slw Wrapper) RowContent() []string {
	n := len(slw.sl.Results)
	return wrapper.RowsFromGetter(n, func(i int) string {
		return slw.content(slw.sl.Results[i], slw.sl.Totals)
	})
}

// TotalRowContent returns all the totals
func (slw Wrapper) TotalRowContent() string {
	return wrapper.TotalRowContent(slw.sl.Totals, slw.content)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (slw Wrapper) EmptyRowContent() string {
	return wrapper.EmptyRowContent(slw.content)
}

// Description describe the stages
func (slw Wrapper) Description() string {
	n := len(slw.sl.Results)
	count := wrapper.CountIf(n, func(i int) bool { return slw.sl.Results[i].SumTimerWait > 0 })
	return fmt.Sprintf("SQL Stage Latency (events_stages_summary_global_by_event_name) %d rows", count)
}

// HaveRelativeStats is true for this object
func (slw Wrapper) HaveRelativeStats() bool {
	return slw.sl.HaveRelativeStats()
}

// FirstCollectTime returns the time the first value was collected
func (slw Wrapper) FirstCollectTime() time.Time {
	return slw.sl.FirstCollected
}

// LastCollectTime returns the time the last value was collected
func (slw Wrapper) LastCollectTime() time.Time {
	return slw.sl.LastCollected
}

// WantRelativeStats indicates if we want relative statistics
func (slw Wrapper) WantRelativeStats() bool {
	return slw.sl.WantRelativeStats()
}

// generate a printable result
func (slw Wrapper) content(row, totals stageslatency.Row) string {
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

// sorting handled inline with sort.Slice to avoid repeated boilerplate types
