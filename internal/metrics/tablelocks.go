package metrics

import (
	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/probe"
	"github.com/sjmudd/ps-top/internal/query"
	"github.com/sjmudd/ps-top/internal/transform"
)

// TableLocks is table lock wait time per table, in total and per lock type,
// from table_lock_waits_summary_by_table.
var TableLocks = Metric{
	Def: metric.Definition{
		Name: "table_locks",
		Columns: []metric.Column{
			{Name: "sum_timer_wait", Kind: metric.Counter},
			{Name: "sum_timer_read", Kind: metric.Counter},
			{Name: "sum_timer_write", Kind: metric.Counter},
			{Name: "sum_timer_read_with_shared_locks", Kind: metric.Counter},
			{Name: "sum_timer_read_high_priority", Kind: metric.Counter},
			{Name: "sum_timer_read_no_insert", Kind: metric.Counter},
			{Name: "sum_timer_read_normal", Kind: metric.Counter},
			{Name: "sum_timer_read_external", Kind: metric.Counter},
			{Name: "sum_timer_write_allow_write", Kind: metric.Counter},
			{Name: "sum_timer_write_concurrent_insert", Kind: metric.Counter},
			{Name: "sum_timer_write_low_priority", Kind: metric.Counter},
			{Name: "sum_timer_write_normal", Kind: metric.Counter},
			{Name: "sum_timer_write_external", Kind: metric.Counter},
		},
		ResetColumn: 0,
	},
	Queries: []probe.Candidate{{
		Name:  "table_lock_waits_summary_by_table",
		Query: `SELECT OBJECT_SCHEMA, OBJECT_NAME, SUM_TIMER_WAIT, SUM_TIMER_READ, SUM_TIMER_WRITE, SUM_TIMER_READ_WITH_SHARED_LOCKS, SUM_TIMER_READ_HIGH_PRIORITY, SUM_TIMER_READ_NO_INSERT, SUM_TIMER_READ_NORMAL, SUM_TIMER_READ_EXTERNAL, SUM_TIMER_WRITE_ALLOW_WRITE, SUM_TIMER_WRITE_CONCURRENT_INSERT, SUM_TIMER_WRITE_LOW_PRIORITY, SUM_TIMER_WRITE_NORMAL, SUM_TIMER_WRITE_EXTERNAL FROM performance_schema.table_lock_waits_summary_by_table WHERE COUNT_STAR > 0`,
	}},
	Columns: []query.Column{
		{Name: "OBJECT_SCHEMA", Type: query.String},
		{Name: "OBJECT_NAME", Type: query.String},
		{Name: "SUM_TIMER_WAIT", Type: query.Uint},
		{Name: "SUM_TIMER_READ", Type: query.Uint},
		{Name: "SUM_TIMER_WRITE", Type: query.Uint},
		{Name: "SUM_TIMER_READ_WITH_SHARED_LOCKS", Type: query.Uint},
		{Name: "SUM_TIMER_READ_HIGH_PRIORITY", Type: query.Uint},
		{Name: "SUM_TIMER_READ_NO_INSERT", Type: query.Uint},
		{Name: "SUM_TIMER_READ_NORMAL", Type: query.Uint},
		{Name: "SUM_TIMER_READ_EXTERNAL", Type: query.Uint},
		{Name: "SUM_TIMER_WRITE_ALLOW_WRITE", Type: query.Uint},
		{Name: "SUM_TIMER_WRITE_CONCURRENT_INSERT", Type: query.Uint},
		{Name: "SUM_TIMER_WRITE_LOW_PRIORITY", Type: query.Uint},
		{Name: "SUM_TIMER_WRITE_NORMAL", Type: query.Uint},
		{Name: "SUM_TIMER_WRITE_EXTERNAL", Type: query.Uint},
	},
	Entry: tableLocksEntry,
}

// tableLocksEntry keys a row by schema.table; values are the 13 timers in
// order.
func tableLocksEntry(r query.Row) transform.Entry {
	values := make([]uint64, 13)
	for i := range values {
		values[i] = r.Uint(2 + i)
	}
	return transform.Entry{Key: r.Str(0) + "." + r.Str(1), Values: values}
}
