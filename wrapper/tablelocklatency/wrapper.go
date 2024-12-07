// Package tablelocklatency holds the routines which manage the file_summary_by_instance table.
package tablelocklatency

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/tablelocks"
	"github.com/sjmudd/ps-top/utils"
)

// Wrapper wraps a TableLockLatency struct
type Wrapper struct {
	tl *tablelocks.TableLocks
}

// NewTableLockLatency creates a wrapper around TableLockLatency
func NewTableLockLatency(cfg *config.Config, db *sql.DB) *Wrapper {
	return &Wrapper{
		tl: tablelocks.NewTableLocks(cfg, db),
	}
}

// ResetStatistics resets the statistics to last values
func (tlw *Wrapper) ResetStatistics() {
	tlw.tl.ResetStatistics()
}

// Collect data from the db, then merge it in.
func (tlw *Wrapper) Collect() {
	tlw.tl.Collect()
	sort.Sort(byLatency(tlw.tl.Results))
}

// Headings returns the headings for a table
func (tlw Wrapper) Headings() string {
	return fmt.Sprintf("%10s %6s|%6s %6s|%6s %6s %6s %6s %6s|%6s %6s %6s %6s %6s|%-30s",
		"Latency", "%",
		"Read", "Write",
		"S.Lock", "High", "NoIns", "Normal", "Extrnl",
		"AlloWr", "CncIns", "Low", "Normal", "Extrnl",
		"Table Name")
}

// RowContent returns the rows we need for displaying
func (tlw Wrapper) RowContent() []string {
	rows := make([]string, 0, len(tlw.tl.Results))

	for i := range tlw.tl.Results {
		rows = append(rows, tlw.content(tlw.tl.Results[i], tlw.tl.Totals))
	}

	return rows
}

// TotalRowContent returns all the totals
func (tlw Wrapper) TotalRowContent() string {
	return tlw.content(tlw.tl.Totals, tlw.tl.Totals)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (tlw Wrapper) EmptyRowContent() string {
	var empty tablelocks.Row

	return tlw.content(empty, empty)
}

// Description returns a description of the table
func (tlw Wrapper) Description() string {
	count := len(tlw.tl.Results)

	return fmt.Sprintf("Locks by Table Name (table_lock_waits_summary_by_table) %d rows", count)
}

// HaveRelativeStats is true for this object
func (tlw Wrapper) HaveRelativeStats() bool {
	return tlw.tl.HaveRelativeStats()
}

// FirstCollectTime returns the time the first value was collected
func (tlw Wrapper) FirstCollectTime() time.Time {
	return tlw.tl.FirstCollected
}

// LastCollectTime returns the time the last value was collected
func (tlw Wrapper) LastCollectTime() time.Time {
	return tlw.tl.LastCollected
}

// WantRelativeStats indiates if we want relative statistics
func (tlw Wrapper) WantRelativeStats() bool {
	return tlw.tl.WantRelativeStats()
}

// content generate a printable result for a row, given the totals
func (tlw Wrapper) content(row, totals tablelocks.Row) string {
	// assume the data is empty so hide it.
	name := row.Name
	if row.SumTimerWait == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s|%6s %6s|%6s %6s %6s %6s %6s|%6s %6s %6s %6s %6s|%s",
		utils.FormatTime(row.SumTimerWait),
		utils.FormatPct(utils.Divide(row.SumTimerWait, totals.SumTimerWait)),

		utils.FormatPct(utils.Divide(row.SumTimerRead, row.SumTimerWait)),
		utils.FormatPct(utils.Divide(row.SumTimerWrite, row.SumTimerWait)),

		utils.FormatPct(utils.Divide(row.SumTimerReadWithSharedLocks, row.SumTimerWait)),
		utils.FormatPct(utils.Divide(row.SumTimerReadHighPriority, row.SumTimerWait)),
		utils.FormatPct(utils.Divide(row.SumTimerReadNoInsert, row.SumTimerWait)),
		utils.FormatPct(utils.Divide(row.SumTimerReadNormal, row.SumTimerWait)),
		utils.FormatPct(utils.Divide(row.SumTimerReadExternal, row.SumTimerWait)),

		utils.FormatPct(utils.Divide(row.SumTimerWriteAllowWrite, row.SumTimerWait)),
		utils.FormatPct(utils.Divide(row.SumTimerWriteConcurrentInsert, row.SumTimerWait)),
		utils.FormatPct(utils.Divide(row.SumTimerWriteLowPriority, row.SumTimerWait)),
		utils.FormatPct(utils.Divide(row.SumTimerWriteNormal, row.SumTimerWait)),
		utils.FormatPct(utils.Divide(row.SumTimerWriteExternal, row.SumTimerWait)),
		name)
}

type byLatency tablelocks.Rows

func (t byLatency) Len() int      { return len(t) }
func (t byLatency) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t byLatency) Less(i, j int) bool {
	return (t[i].SumTimerWait > t[j].SumTimerWait) ||
		((t[i].SumTimerWait == t[j].SumTimerWait) &&
			(t[i].Name < t[j].Name))

}
