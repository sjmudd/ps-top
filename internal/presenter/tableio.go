package presenter

import (
	"github.com/sjmudd/ps-top/internal/formatter"
	"github.com/sjmudd/ps-top/internal/metrics"
)

// TableIOLatency shows table I/O wait time per table.
var TableIOLatency = View{
	Title: "Table I/O Latency (table_io_waits_summary_by_table)",
	Def:   metrics.TableIO.Def,
	Cells: [][]Cell{
		{Value("Latency", "sum_timer_wait", formatter.Time, 10), ShareOfTotal("%", "sum_timer_wait", 6)},
		{ShareOf("Read", "sum_timer_read", "sum_timer_wait", 6), ShareOf("Write", "sum_timer_write", "sum_timer_wait", 6)},
		{ShareOf("Fetch", "sum_timer_fetch", "sum_timer_wait", 6), ShareOf("Insert", "sum_timer_insert", "sum_timer_wait", 6),
			ShareOf("Update", "sum_timer_update", "sum_timer_wait", 6), ShareOf("Delete", "sum_timer_delete", "sum_timer_wait", 6)},
		{Key("Table Name", 0)},
	},
	SortColumn: "sum_timer_wait",
}

// TableIOOps shows table I/O operation counts per table.
var TableIOOps = View{
	Title: "Table I/O Ops (table_io_waits_summary_by_table)",
	Def:   metrics.TableIO.Def,
	Cells: [][]Cell{
		{Value("Ops", "count_star", formatter.Count, 10), ShareOfTotal("%", "count_star", 6)},
		{ShareOf("Read", "count_read", "count_star", 6), ShareOf("Write", "count_write", "count_star", 6)},
		{ShareOf("Fetch", "count_fetch", "count_star", 6), ShareOf("Insert", "count_insert", "count_star", 6),
			ShareOf("Update", "count_update", "count_star", 6), ShareOf("Delete", "count_delete", "count_star", 6)},
		{Key("Table Name", 0)},
	},
	SortColumn: "count_star",
}
