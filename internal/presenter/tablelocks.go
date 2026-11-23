package presenter

import (
	"github.com/sjmudd/ps-top/internal/formatter"
	"github.com/sjmudd/ps-top/internal/metrics/tablelocks"
)

// TableLockLatency shows table lock wait time per table, broken down by
// lock type as a share of the table's wait time.
var TableLockLatency = View[tablelocks.Column]{
	Name:      "table_lock_latency",
	Title:     "Locks by Table Name",
	Spec:      tablelocks.Spec,
	Anonymise: anonymiseTable,
	Cells: [][]Cell[tablelocks.Column]{
		{Value("Latency", tablelocks.SumTimerWait, formatter.Time, 10), ShareOfTotal("%", tablelocks.SumTimerWait, 6)},
		{ShareOf("Read", tablelocks.SumTimerRead, tablelocks.SumTimerWait, 6), ShareOf("Write", tablelocks.SumTimerWrite, tablelocks.SumTimerWait, 6)},
		{ShareOf("S.Lock", tablelocks.SumTimerReadWithSharedLocks, tablelocks.SumTimerWait, 6),
			ShareOf("High", tablelocks.SumTimerReadHighPriority, tablelocks.SumTimerWait, 6),
			ShareOf("NoIns", tablelocks.SumTimerReadNoInsert, tablelocks.SumTimerWait, 6),
			ShareOf("Normal", tablelocks.SumTimerReadNormal, tablelocks.SumTimerWait, 6),
			ShareOf("Extrnl", tablelocks.SumTimerReadExternal, tablelocks.SumTimerWait, 6)},
		{ShareOf("AlloWr", tablelocks.SumTimerWriteAllowWrite, tablelocks.SumTimerWait, 6),
			ShareOf("CncIns", tablelocks.SumTimerWriteConcurrentInsert, tablelocks.SumTimerWait, 6),
			ShareOf("Low", tablelocks.SumTimerWriteLowPriority, tablelocks.SumTimerWait, 6),
			ShareOf("Normal", tablelocks.SumTimerWriteNormal, tablelocks.SumTimerWait, 6),
			ShareOf("Extrnl", tablelocks.SumTimerWriteExternal, tablelocks.SumTimerWait, 6)},
		{Key[tablelocks.Column]("Table Name", 0)},
	},
	SortColumn: tablelocks.SumTimerWait,
}
