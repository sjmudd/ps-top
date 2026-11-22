package presenter

import (
	"strings"

	"github.com/sjmudd/ps-top/internal/formatter"
	"github.com/sjmudd/ps-top/internal/metrics/stages"
)

// StagesLatency shows time and count per statement execution stage, SQL
// stages named without the "stage/sql/" prefix, e.g. "Sending data".
var StagesLatency = View[stages.Column]{
	Name:   "stages_latency",
	Title:  "SQL Stage Latency",
	Spec:   stages.Spec,
	MapKey: func(k string) string { return strings.TrimPrefix(k, "stage/sql/") },
	Cells: [][]Cell[stages.Column]{
		{Value("Latency", stages.SumTimerWait, formatter.Time, 10), ShareOfTotal("%", stages.SumTimerWait, 6),
			Value("Counter", stages.CountStar, formatter.Amount, 8)},
		{Key[stages.Column]("Stage Name", 0)},
	},
	SortColumn: stages.SumTimerWait,
}
