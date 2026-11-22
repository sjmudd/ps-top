// Package stages is time spent per statement execution stage, from
// events_stages_summary_global_by_event_name. A row's key is the event name,
// e.g. stage/sql/Sending data.
package stages

import "github.com/sjmudd/ps-top/internal/metric"

// Column is a value column of Spec.
type Column int

// The value columns, in Spec.Columns order.
const (
	CountStar Column = iota
	SumTimerWait
)

// Spec describes the metric.
var Spec = metric.Spec[Column]{
	Name:   "stages",
	Tables: []string{"performance_schema.events_stages_summary_global_by_event_name"},
	Keys:   []string{"EVENT_NAME"},
	Columns: []metric.Column{
		CountStar:    {Name: "COUNT_STAR", Kind: metric.Counter},
		SumTimerWait: {Name: "SUM_TIMER_WAIT", Kind: metric.Counter},
	},
	Where:       "COUNT_STAR > 0",
	Types:       []metric.MetricType{metric.DifferenceValues},
	ResetColumn: SumTimerWait,
}
