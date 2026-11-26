// Package tablelocks is table lock wait time per table, in total and per lock
// type, from table_lock_waits_summary_by_table. A row's key is schema.table.
package tablelocks

import "github.com/sjmudd/ps-top/v2/internal/metric"

// Column is a value column of Spec.
type Column int

// The value columns, in Spec.Columns order.
const (
	SumTimerWait Column = iota
	SumTimerRead
	SumTimerWrite
	SumTimerReadWithSharedLocks
	SumTimerReadHighPriority
	SumTimerReadNoInsert
	SumTimerReadNormal
	SumTimerReadExternal
	SumTimerWriteAllowWrite
	SumTimerWriteConcurrentInsert
	SumTimerWriteLowPriority
	SumTimerWriteNormal
	SumTimerWriteExternal
)

// Spec describes the metric.
var Spec = metric.Spec[Column]{
	Name:   "table_locks",
	Tables: []string{"performance_schema.table_lock_waits_summary_by_table"},
	Keys:   []string{"OBJECT_SCHEMA", "OBJECT_NAME"},
	Columns: []metric.Column{
		SumTimerWait:                  {Name: "SUM_TIMER_WAIT", Kind: metric.Counter},
		SumTimerRead:                  {Name: "SUM_TIMER_READ", Kind: metric.Counter},
		SumTimerWrite:                 {Name: "SUM_TIMER_WRITE", Kind: metric.Counter},
		SumTimerReadWithSharedLocks:   {Name: "SUM_TIMER_READ_WITH_SHARED_LOCKS", Kind: metric.Counter},
		SumTimerReadHighPriority:      {Name: "SUM_TIMER_READ_HIGH_PRIORITY", Kind: metric.Counter},
		SumTimerReadNoInsert:          {Name: "SUM_TIMER_READ_NO_INSERT", Kind: metric.Counter},
		SumTimerReadNormal:            {Name: "SUM_TIMER_READ_NORMAL", Kind: metric.Counter},
		SumTimerReadExternal:          {Name: "SUM_TIMER_READ_EXTERNAL", Kind: metric.Counter},
		SumTimerWriteAllowWrite:       {Name: "SUM_TIMER_WRITE_ALLOW_WRITE", Kind: metric.Counter},
		SumTimerWriteConcurrentInsert: {Name: "SUM_TIMER_WRITE_CONCURRENT_INSERT", Kind: metric.Counter},
		SumTimerWriteLowPriority:      {Name: "SUM_TIMER_WRITE_LOW_PRIORITY", Kind: metric.Counter},
		SumTimerWriteNormal:           {Name: "SUM_TIMER_WRITE_NORMAL", Kind: metric.Counter},
		SumTimerWriteExternal:         {Name: "SUM_TIMER_WRITE_EXTERNAL", Kind: metric.Counter},
	},
	Where:       "COUNT_STAR > 0",
	Schema:      "OBJECT_SCHEMA",
	Types:       []metric.MetricType{metric.DifferenceValues, metric.CurrentValue},
	ResetColumn: SumTimerWait,
}
