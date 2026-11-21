package presenter

import (
	"github.com/sjmudd/ps-top/internal/formatter"
	"github.com/sjmudd/ps-top/internal/metrics"
)

// TableLockLatency shows table lock wait time per table, broken down by
// lock type as a share of the table's wait time.
var TableLockLatency = View{
	Title: "Locks by Table Name (table_lock_waits_summary_by_table)",
	Def:   metrics.TableLocks.Def,
	Cells: [][]Cell{
		{Value("Latency", "sum_timer_wait", formatter.Time, 10), ShareOfTotal("%", "sum_timer_wait", 6)},
		{ShareOf("Read", "sum_timer_read", "sum_timer_wait", 6), ShareOf("Write", "sum_timer_write", "sum_timer_wait", 6)},
		{ShareOf("S.Lock", "sum_timer_read_with_shared_locks", "sum_timer_wait", 6),
			ShareOf("High", "sum_timer_read_high_priority", "sum_timer_wait", 6),
			ShareOf("NoIns", "sum_timer_read_no_insert", "sum_timer_wait", 6),
			ShareOf("Normal", "sum_timer_read_normal", "sum_timer_wait", 6),
			ShareOf("Extrnl", "sum_timer_read_external", "sum_timer_wait", 6)},
		{ShareOf("AlloWr", "sum_timer_write_allow_write", "sum_timer_wait", 6),
			ShareOf("CncIns", "sum_timer_write_concurrent_insert", "sum_timer_wait", 6),
			ShareOf("Low", "sum_timer_write_low_priority", "sum_timer_wait", 6),
			ShareOf("Normal", "sum_timer_write_normal", "sum_timer_wait", 6),
			ShareOf("Extrnl", "sum_timer_write_external", "sum_timer_wait", 6)},
		{Key("Table Name", 0)},
	},
	SortColumn: "sum_timer_wait",
}
