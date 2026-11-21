package presenter

import (
	"github.com/sjmudd/ps-top/internal/formatter"
	"github.com/sjmudd/ps-top/internal/metrics"
)

// FileIOLatency shows file I/O wait time, bytes and operations per file,
// with file names mapped to tables or labels by mapKey (built from the
// server's settings, so the view is made after the probe).
func FileIOLatency(mapKey func(string) string) View {
	return View{
		Title:  "File I/O Latency (file_summary_by_instance)",
		Def:    metrics.FileIO.Def,
		MapKey: mapKey,
		Cells: [][]Cell{
			{Value("Latency", "sum_timer_wait", formatter.Time, 10), ShareOfTotal("%", "sum_timer_wait")},
			{ShareOf("Read", "sum_timer_read", "sum_timer_wait"), ShareOf("Write", "sum_timer_write", "sum_timer_wait"),
				ShareOf("Misc", "sum_timer_misc", "sum_timer_wait")},
			{Value("Rd bytes", "sum_number_of_bytes_read", formatter.Amount, 8),
				Value("Wr bytes", "sum_number_of_bytes_write", formatter.Amount, 8)},
			{Value("Ops", "count_star", formatter.Amount, 8), ShareOf("R Ops", "count_read", "count_star"),
				ShareOf("W Ops", "count_write", "count_star"), ShareOf("M Ops", "count_misc", "count_star")},
			{Key("Table Name", 0)},
		},
		SortColumn: "sum_timer_wait",
	}
}
