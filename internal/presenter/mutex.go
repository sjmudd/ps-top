package presenter

import (
	"strings"

	"github.com/sjmudd/ps-top/internal/formatter"
	"github.com/sjmudd/ps-top/internal/metrics"
)

// MutexLatency shows wait time and count per mutex or InnoDB sx-lock,
// named without the "wait/synch/" prefix, e.g. mutex/innodb/trx_sys_mutex.
var MutexLatency = View{
	Title:  "Mutex Latency (events_waits_summary_global_by_event_name)",
	Def:    metrics.SynchWaits.Def,
	MapKey: func(k string) string { return strings.TrimPrefix(k, "wait/synch/") },
	Cells: [][]Cell{
		{Value("Latency", "sum_timer_wait", formatter.Time, 10), Value("MtxCnt", "count_star", formatter.Amount, 8),
			ShareOfTotal("%", "sum_timer_wait", 8)},
		{Key("Mutex Name", 0)},
	},
	SortColumn: "sum_timer_wait",
}
