package metrics

import (
	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/probe"
	"github.com/sjmudd/ps-top/internal/query"
	"github.com/sjmudd/ps-top/internal/transform"
)

// FileIO is file I/O per file name: operation counts, wait times and bytes,
// in total and per operation, from file_summary_by_instance.
// SUM_NUMBER_OF_BYTES_* are signed bigint on every version; sums of bytes
// are never negative, so they are read as unsigned, as the old code does.
var FileIO = Metric{
	Def: metric.Definition{
		Name: "file_io",
		Columns: []metric.Column{
			{Name: "count_star", Kind: metric.Counter},
			{Name: "sum_timer_wait", Kind: metric.Counter},
			{Name: "count_read", Kind: metric.Counter},
			{Name: "sum_timer_read", Kind: metric.Counter},
			{Name: "count_write", Kind: metric.Counter},
			{Name: "sum_timer_write", Kind: metric.Counter},
			{Name: "count_misc", Kind: metric.Counter},
			{Name: "sum_timer_misc", Kind: metric.Counter},
			{Name: "sum_number_of_bytes_read", Kind: metric.Counter},
			{Name: "sum_number_of_bytes_write", Kind: metric.Counter},
		},
		ResetColumn: 1,
	},
	Queries: []probe.Candidate{{
		Name:  "file_summary_by_instance",
		Query: `SELECT FILE_NAME, COUNT_STAR, SUM_TIMER_WAIT, COUNT_READ, SUM_TIMER_READ, COUNT_WRITE, SUM_TIMER_WRITE, COUNT_MISC, SUM_TIMER_MISC, SUM_NUMBER_OF_BYTES_READ, SUM_NUMBER_OF_BYTES_WRITE FROM performance_schema.file_summary_by_instance WHERE COUNT_STAR > 0`,
	}},
	Columns: []query.Column{
		{Name: "FILE_NAME", Type: query.String},
		{Name: "COUNT_STAR", Type: query.Uint},
		{Name: "SUM_TIMER_WAIT", Type: query.Uint},
		{Name: "COUNT_READ", Type: query.Uint},
		{Name: "SUM_TIMER_READ", Type: query.Uint},
		{Name: "COUNT_WRITE", Type: query.Uint},
		{Name: "SUM_TIMER_WRITE", Type: query.Uint},
		{Name: "COUNT_MISC", Type: query.Uint},
		{Name: "SUM_TIMER_MISC", Type: query.Uint},
		{Name: "SUM_NUMBER_OF_BYTES_READ", Type: query.Uint},
		{Name: "SUM_NUMBER_OF_BYTES_WRITE", Type: query.Uint},
	},
	Entry: fileIOEntry,
}

// fileIOEntry keys a row by file name; values are the 10 columns in order.
func fileIOEntry(r query.Row) transform.Entry {
	values := make([]uint64, 10)
	for i := range values {
		values[i] = r.Uint(1 + i)
	}
	return transform.Entry{Key: r.Str(0), Values: values}
}
