package metrics

import (
	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/probe"
	"github.com/sjmudd/ps-top/internal/query"
	"github.com/sjmudd/ps-top/internal/transform"
)

// TableIO is table I/O per table: operation counts and wait times, in total
// and per operation, from table_io_waits_summary_by_table.
var TableIO = Metric{
	Def: metric.Definition{
		Name: "table_io",
		Columns: []metric.Column{
			{Name: "count_star", Kind: metric.Counter},
			{Name: "sum_timer_wait", Kind: metric.Counter},
			{Name: "count_read", Kind: metric.Counter},
			{Name: "sum_timer_read", Kind: metric.Counter},
			{Name: "count_write", Kind: metric.Counter},
			{Name: "sum_timer_write", Kind: metric.Counter},
			{Name: "count_fetch", Kind: metric.Counter},
			{Name: "sum_timer_fetch", Kind: metric.Counter},
			{Name: "count_insert", Kind: metric.Counter},
			{Name: "sum_timer_insert", Kind: metric.Counter},
			{Name: "count_update", Kind: metric.Counter},
			{Name: "sum_timer_update", Kind: metric.Counter},
			{Name: "count_delete", Kind: metric.Counter},
			{Name: "sum_timer_delete", Kind: metric.Counter},
		},
		ResetColumn: 1,
	},
	Queries: []probe.Candidate{{
		Name:  "table_io_waits_summary_by_table",
		Query: `SELECT OBJECT_SCHEMA, OBJECT_NAME, COUNT_STAR, SUM_TIMER_WAIT, COUNT_READ, SUM_TIMER_READ, COUNT_WRITE, SUM_TIMER_WRITE, COUNT_FETCH, SUM_TIMER_FETCH, COUNT_INSERT, SUM_TIMER_INSERT, COUNT_UPDATE, SUM_TIMER_UPDATE, COUNT_DELETE, SUM_TIMER_DELETE FROM performance_schema.table_io_waits_summary_by_table WHERE COUNT_STAR > 0`,
	}},
	Columns: []query.Column{
		{Name: "OBJECT_SCHEMA", Type: query.String},
		{Name: "OBJECT_NAME", Type: query.String},
		{Name: "COUNT_STAR", Type: query.Uint},
		{Name: "SUM_TIMER_WAIT", Type: query.Uint},
		{Name: "COUNT_READ", Type: query.Uint},
		{Name: "SUM_TIMER_READ", Type: query.Uint},
		{Name: "COUNT_WRITE", Type: query.Uint},
		{Name: "SUM_TIMER_WRITE", Type: query.Uint},
		{Name: "COUNT_FETCH", Type: query.Uint},
		{Name: "SUM_TIMER_FETCH", Type: query.Uint},
		{Name: "COUNT_INSERT", Type: query.Uint},
		{Name: "SUM_TIMER_INSERT", Type: query.Uint},
		{Name: "COUNT_UPDATE", Type: query.Uint},
		{Name: "SUM_TIMER_UPDATE", Type: query.Uint},
		{Name: "COUNT_DELETE", Type: query.Uint},
		{Name: "SUM_TIMER_DELETE", Type: query.Uint},
	},
	Entry: tableIOEntry,
}

// tableIOEntry keys a row by schema.table; values are the 14 counters in order.
func tableIOEntry(r query.Row) transform.Entry {
	values := make([]uint64, 14)
	for i := range values {
		values[i] = r.Uint(2 + i)
	}
	return transform.Entry{Key: r.Str(0) + "." + r.Str(1), Values: values}
}
