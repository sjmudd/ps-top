package presenter

import (
	"github.com/sjmudd/ps-top/internal/formatter"
	"github.com/sjmudd/ps-top/internal/metrics"
)

// TableIOLatency shows table I/O wait time per table.
var TableIOLatency = View{
	Title:   "Table I/O Latency (table_io_waits_summary_by_table)",
	Def:     metrics.TableIO.Def,
	Heading: "Latency",
	Total:   "sum_timer_wait",
	Format:  formatter.Time,
	Groups: [][]Part{
		{{"Read", "sum_timer_read"}, {"Write", "sum_timer_write"}},
		{{"Fetch", "sum_timer_fetch"}, {"Insert", "sum_timer_insert"}, {"Update", "sum_timer_update"}, {"Delete", "sum_timer_delete"}},
	},
	KeyHeading: "Table Name",
}

// TableIOOps shows table I/O operation counts per table.
var TableIOOps = View{
	Title:   "Table I/O Ops (table_io_waits_summary_by_table)",
	Def:     metrics.TableIO.Def,
	Heading: "Ops",
	Total:   "count_star",
	Format:  func(c uint64) string { return formatter.Count(c, 10) },
	Groups: [][]Part{
		{{"Read", "count_read"}, {"Write", "count_write"}},
		{{"Fetch", "count_fetch"}, {"Insert", "count_insert"}, {"Update", "count_update"}, {"Delete", "count_delete"}},
	},
	KeyHeading: "Table Name",
}
