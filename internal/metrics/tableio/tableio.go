// Package tableio is table I/O per table: operation counts and wait times, in
// total and per operation, from table_io_waits_summary_by_table. A row's key
// is schema.table.
package tableio

import "github.com/sjmudd/ps-top/internal/metric"

// Column is a value column of Spec.
type Column int

// The value columns, in Spec.Columns order.
const (
	CountStar Column = iota
	SumTimerWait
	CountRead
	SumTimerRead
	CountWrite
	SumTimerWrite
	CountFetch
	SumTimerFetch
	CountInsert
	SumTimerInsert
	CountUpdate
	SumTimerUpdate
	CountDelete
	SumTimerDelete
)

// Spec describes the metric.
var Spec = metric.Spec[Column]{
	Name:   "table_io",
	Tables: []string{"performance_schema.table_io_waits_summary_by_table"},
	Keys:   []string{"OBJECT_SCHEMA", "OBJECT_NAME"},
	Columns: []metric.Column{
		CountStar:      {Name: "COUNT_STAR", Kind: metric.Counter},
		SumTimerWait:   {Name: "SUM_TIMER_WAIT", Kind: metric.Counter},
		CountRead:      {Name: "COUNT_READ", Kind: metric.Counter},
		SumTimerRead:   {Name: "SUM_TIMER_READ", Kind: metric.Counter},
		CountWrite:     {Name: "COUNT_WRITE", Kind: metric.Counter},
		SumTimerWrite:  {Name: "SUM_TIMER_WRITE", Kind: metric.Counter},
		CountFetch:     {Name: "COUNT_FETCH", Kind: metric.Counter},
		SumTimerFetch:  {Name: "SUM_TIMER_FETCH", Kind: metric.Counter},
		CountInsert:    {Name: "COUNT_INSERT", Kind: metric.Counter},
		SumTimerInsert: {Name: "SUM_TIMER_INSERT", Kind: metric.Counter},
		CountUpdate:    {Name: "COUNT_UPDATE", Kind: metric.Counter},
		SumTimerUpdate: {Name: "SUM_TIMER_UPDATE", Kind: metric.Counter},
		CountDelete:    {Name: "COUNT_DELETE", Kind: metric.Counter},
		SumTimerDelete: {Name: "SUM_TIMER_DELETE", Kind: metric.Counter},
	},
	Where:       "COUNT_STAR > 0",
	Schema:      "OBJECT_SCHEMA",
	Types:       []metric.MetricType{metric.DifferenceValues, metric.CurrentValue},
	ResetColumn: SumTimerWait,
}
