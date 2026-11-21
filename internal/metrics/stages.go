package metrics

import (
	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/probe"
	"github.com/sjmudd/ps-top/internal/query"
)

// Stages is time spent per statement execution stage (event name, e.g.
// stage/sql/Sending data), from events_stages_summary_global_by_event_name.
var Stages = Metric{
	Def: metric.Definition{
		Name: "stages",
		Columns: []metric.Column{
			{Name: "count_star", Kind: metric.Counter},
			{Name: "sum_timer_wait", Kind: metric.Counter},
		},
		ResetColumn: 1,
	},
	Queries: []probe.Candidate{{
		Name:  "events_stages_summary_global_by_event_name",
		Query: `SELECT EVENT_NAME, COUNT_STAR, SUM_TIMER_WAIT FROM performance_schema.events_stages_summary_global_by_event_name WHERE COUNT_STAR > 0`,
	}},
	Columns: []query.Column{
		{Name: "EVENT_NAME", Type: query.String},
		{Name: "COUNT_STAR", Type: query.Uint},
		{Name: "SUM_TIMER_WAIT", Type: query.Uint},
	},
	Entry: eventEntry,
}
