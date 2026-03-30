// Package tablelocklatency holds the routines which manage the file_summary_by_instance table.
package tablelocklatency

import (
	"database/sql"
	"fmt"
	"slices"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/tablelocks"
	"github.com/sjmudd/ps-top/utils"
	"github.com/sjmudd/ps-top/wrapper"
)

var (
	defaultSort = func(rows []tablelocks.Row) {
		slices.SortFunc(rows, func(a, b tablelocks.Row) int {
			return utils.SumTimerWaitNameOrdering(
				utils.NewSumTimerWaitName(a.Name, a.SumTimerWait),
				utils.NewSumTimerWaitName(b.Name, b.SumTimerWait),
			)
		})
	}

	// No hasData filter; count all rows.
	// We'll not define a variable and pass nil directly.

	defaultContent = func(row, totals tablelocks.Row) string {
		name := row.Name
		if row.SumTimerWait == 0 && name != "Totals" {
			name = ""
		}
		timeStr, pctStr := wrapper.TimePct(row.SumTimerWait, totals.SumTimerWait)
		pct := wrapper.PctStrings(row.SumTimerWait,
			row.SumTimerRead,
			row.SumTimerWrite,
			row.SumTimerReadWithSharedLocks,
			row.SumTimerReadHighPriority,
			row.SumTimerReadNoInsert,
			row.SumTimerReadNormal,
			row.SumTimerReadExternal,
			row.SumTimerWriteAllowWrite,
			row.SumTimerWriteConcurrentInsert,
			row.SumTimerWriteLowPriority,
			row.SumTimerWriteNormal,
			row.SumTimerWriteExternal)

		return fmt.Sprintf("%10s %6s|%6s %6s|%6s %6s %6s %6s %6s|%6s %6s %6s %6s %6s|%s",
			timeStr,
			pctStr,

			pct[0],
			pct[1],

			pct[2],
			pct[3],
			pct[4],
			pct[5],
			pct[6],

			pct[7],
			pct[8],
			pct[9],
			pct[10],
			pct[11],
			name)
	}
)

// Wrapper wraps a TableLock struct.
type Wrapper struct {
	*wrapper.BaseWrapper[tablelocks.Row, *tablelocks.TableLocks]
}

// NewTableLockLatency creates a wrapper around TableLockLatency.
func NewTableLockLatency(cfg *config.Config, db *sql.DB) *Wrapper {
	tl := tablelocks.NewTableLocks(cfg, db)
	bw := wrapper.NewBaseWrapper(
		tl,
		"Locks by Table Name (table_lock_waits_summary_by_table)",
		defaultSort,
		nil, // hasData: count all rows
		defaultContent,
	)
	return &Wrapper{BaseWrapper: bw}
}

// Headings returns the headings for a table.
func (w *Wrapper) Headings() string {
	return fmt.Sprintf("%10s %6s|%6s %6s|%6s %6s %6s %6s %6s|%6s %6s %6s %6s %6s|%-30s",
		"Latency", "%",
		"Read", "Write",
		"S.Lock", "High", "NoIns", "Normal", "Extrnl",
		"AlloWr", "CncIns", "Low", "Normal", "Extrnl",
		"Table Name")
}
