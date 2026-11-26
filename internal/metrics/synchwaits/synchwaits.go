// Package synchwaits is waits on server mutexes and InnoDB sx-locks per
// instrument, from events_waits_summary_global_by_event_name. A row's key is
// the event name, e.g. wait/synch/mutex/innodb/trx_sys_mutex.
package synchwaits

import "github.com/sjmudd/ps-top/v2/internal/metric"

// Column is a value column of Spec.
type Column int

// The value columns, in Spec.Columns order.
const (
	CountStar Column = iota
	SumTimerWait
)

// Spec describes the metric.
var Spec = metric.Spec[Column]{
	Name:   "synch_waits",
	Tables: []string{"performance_schema.events_waits_summary_global_by_event_name"},
	Keys:   []string{"EVENT_NAME"},
	Columns: []metric.Column{
		CountStar:    {Name: "COUNT_STAR", Kind: metric.Counter},
		SumTimerWait: {Name: "SUM_TIMER_WAIT", Kind: metric.Counter},
	},
	Where:       "COUNT_STAR > 0 AND (EVENT_NAME LIKE 'wait/synch/mutex/%' OR EVENT_NAME LIKE 'wait/synch/sxlock/innodb/%')",
	Types:       []metric.MetricType{metric.DifferenceValues, metric.CurrentValue},
	ResetColumn: SumTimerWait,
}
