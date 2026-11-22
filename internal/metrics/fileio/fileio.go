// Package fileio is file I/O per file name: operation counts, wait times and
// bytes, in total and per operation, from file_summary_by_instance. A row's
// key is the file name as the server reports it.
//
// SUM_NUMBER_OF_BYTES_* are signed bigint on every version. Sums of bytes are
// never negative, so they are read as unsigned, as the old code does.
package fileio

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
	CountMisc
	SumTimerMisc
	SumNumberOfBytesRead
	SumNumberOfBytesWrite
)

// Spec describes the metric.
var Spec = metric.Spec[Column]{
	Name:   "file_io",
	Tables: []string{"performance_schema.file_summary_by_instance"},
	Keys:   []string{"FILE_NAME"},
	Columns: []metric.Column{
		CountStar:             {Name: "COUNT_STAR", Kind: metric.Counter},
		SumTimerWait:          {Name: "SUM_TIMER_WAIT", Kind: metric.Counter},
		CountRead:             {Name: "COUNT_READ", Kind: metric.Counter},
		SumTimerRead:          {Name: "SUM_TIMER_READ", Kind: metric.Counter},
		CountWrite:            {Name: "COUNT_WRITE", Kind: metric.Counter},
		SumTimerWrite:         {Name: "SUM_TIMER_WRITE", Kind: metric.Counter},
		CountMisc:             {Name: "COUNT_MISC", Kind: metric.Counter},
		SumTimerMisc:          {Name: "SUM_TIMER_MISC", Kind: metric.Counter},
		SumNumberOfBytesRead:  {Name: "SUM_NUMBER_OF_BYTES_READ", Kind: metric.Counter},
		SumNumberOfBytesWrite: {Name: "SUM_NUMBER_OF_BYTES_WRITE", Kind: metric.Counter},
	},
	Where:       "COUNT_STAR > 0",
	Types:       []metric.MetricType{metric.DifferenceValues, metric.CurrentValue},
	ResetColumn: SumTimerWait,
}
