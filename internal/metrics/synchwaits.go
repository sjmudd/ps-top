package metrics

import (
	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/probe"
	"github.com/sjmudd/ps-top/internal/query"
	"github.com/sjmudd/ps-top/internal/transform"
)

// SynchWaits is waits on server mutexes and InnoDB sx-locks per instrument
// (event name, e.g. wait/synch/mutex/innodb/trx_sys_mutex), from
// events_waits_summary_global_by_event_name.
var SynchWaits = Metric{
	Def: metric.Definition{
		Name: "synch_waits",
		Columns: []metric.Column{
			{Name: "count_star", Kind: metric.Counter},
			{Name: "sum_timer_wait", Kind: metric.Counter},
		},
		ResetColumn: 1,
	},
	Queries: []probe.Candidate{{
		Name:  "events_waits_summary_global_by_event_name",
		Query: `SELECT EVENT_NAME, COUNT_STAR, SUM_TIMER_WAIT FROM performance_schema.events_waits_summary_global_by_event_name WHERE COUNT_STAR > 0 AND (EVENT_NAME LIKE 'wait/synch/mutex/%' OR EVENT_NAME LIKE 'wait/synch/sxlock/innodb/%')`,
	}},
	Columns: []query.Column{
		{Name: "EVENT_NAME", Type: query.String},
		{Name: "COUNT_STAR", Type: query.Uint},
		{Name: "SUM_TIMER_WAIT", Type: query.Uint},
	},
	Entry: eventEntry,
}

// eventEntry keys a row by event name; values are COUNT_STAR and
// SUM_TIMER_WAIT.
func eventEntry(r query.Row) transform.Entry {
	return transform.Entry{Key: r.Str(0), Values: []uint64{r.Uint(1), r.Uint(2)}}
}
