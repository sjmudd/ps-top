package presenter

import (
	"strings"

	"github.com/sjmudd/ps-top/internal/formatter"
	"github.com/sjmudd/ps-top/internal/metrics/synchwaits"
)

// MutexLatency shows wait time and count per mutex or InnoDB sx-lock,
// named without the "wait/synch/" prefix, e.g. mutex/innodb/trx_sys_mutex.
var MutexLatency = View[synchwaits.Column]{
	Name:   "mutex_latency",
	Title:  "Mutex Latency",
	Spec:   synchwaits.Spec,
	MapKey: func(k string) string { return strings.TrimPrefix(k, "wait/synch/") },
	Cells: [][]Cell[synchwaits.Column]{
		{Value("Latency", synchwaits.SumTimerWait, formatter.Time, 10), Value("MtxCnt", synchwaits.CountStar, formatter.Amount, 8),
			ShareOfTotal("%", synchwaits.SumTimerWait, 8)},
		{Key[synchwaits.Column]("Mutex Name", 0)},
	},
	SortColumn: synchwaits.SumTimerWait,
}
