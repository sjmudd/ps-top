package presenter

import (
	"strings"

	"github.com/sjmudd/ps-top/internal/formatter"
	"github.com/sjmudd/ps-top/internal/metrics"
)

// StagesLatency shows time and count per statement execution stage, SQL
// stages named without the "stage/sql/" prefix, e.g. "Sending data".
var StagesLatency = View{
	Title:  "SQL Stage Latency (events_stages_summary_global_by_event_name)",
	Def:    metrics.Stages.Def,
	MapKey: func(k string) string { return strings.TrimPrefix(k, "stage/sql/") },
	Cells: [][]Cell{
		{Value("Latency", "sum_timer_wait", formatter.Time, 10), ShareOfTotal("%", "sum_timer_wait", 6),
			Value("Counter", "count_star", formatter.Amount, 8)},
		{Key("Stage Name", 0)},
	},
	SortColumn: "sum_timer_wait",
}
