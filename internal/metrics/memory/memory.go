// Package memory is memory in use per instrument, from
// memory_summary_global_by_event_name: current and highest allocation counts
// and bytes, and the number of allocations and frees. A row's key is the
// event name, e.g. memory/innodb/buf_buf_pool. Only current values are shown
// (CurrentValue), so no snapshot is subtracted from another.
package memory

import "github.com/sjmudd/ps-top/internal/metric"

// Column is a value column of Spec.
type Column int

// The value columns, in Spec.Columns order.
//
// CURRENT_* and HIGH_* are signed bigint. They can be negative on MySQL, as
// memory freed by a thread other than the one that allocated it is counted
// against the freeing thread. They are read as unsigned, so a negative value
// fails to scan and stops collection. Not seen so far. If it happens, this
// needs fixing, e.g. by reading signed values.
const (
	CurrentCountUsed Column = iota
	HighCountUsed
	CurrentNumberOfBytesUsed
	HighNumberOfBytesUsed
	MemOps // COUNT_ALLOC + COUNT_FREE
)

// Spec describes the metric.
var Spec = metric.Spec[Column]{
	Name:  "memory",
	Table: "memory_summary_global_by_event_name",
	Keys:  []string{"EVENT_NAME"},
	Columns: []metric.Column{
		CurrentCountUsed:         {Name: "CURRENT_COUNT_USED", Kind: metric.Gauge},
		HighCountUsed:            {Name: "HIGH_COUNT_USED", Kind: metric.Gauge},
		CurrentNumberOfBytesUsed: {Name: "CURRENT_NUMBER_OF_BYTES_USED", Kind: metric.Gauge},
		HighNumberOfBytesUsed:    {Name: "HIGH_NUMBER_OF_BYTES_USED", Kind: metric.Gauge},
		MemOps:                   {Name: "MEM_OPS", Kind: metric.Counter, Expr: "COUNT_ALLOC + COUNT_FREE"},
	},
	Where: "HIGH_COUNT_USED > 0",
	Types: []metric.MetricType{metric.CurrentValue},
}
