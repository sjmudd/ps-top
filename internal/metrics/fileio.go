package metrics

import (
	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/probe"
	"github.com/sjmudd/ps-top/internal/query"
	"github.com/sjmudd/ps-top/internal/transform"
)

// fileIO is file I/O per file name.
var fileIO = Metric{
	Def: metric.Definition{
		Name: "file_io",
		Columns: []metric.Column{
			{Name: "count_star", Kind: metric.Counter},
			{Name: "sum_timer_wait", Kind: metric.Counter},
		},
		ResetColumn: 1,
	},
	Queries: []probe.Candidate{{
		Name:  "file_summary_by_instance",
		Query: `SELECT FILE_NAME, COUNT_STAR, SUM_TIMER_WAIT FROM performance_schema.file_summary_by_instance WHERE COUNT_STAR > 0`,
	}},
	Columns: []query.Column{
		{Name: "FILE_NAME", Type: query.String},
		{Name: "COUNT_STAR", Type: query.Uint},
		{Name: "SUM_TIMER_WAIT", Type: query.Uint},
	},
	Entry: fileIOEntry,
}

func fileIOEntry(r query.Row) transform.Entry {
	return transform.Entry{Key: r.Str(0), Values: []uint64{r.Uint(1), r.Uint(2)}}
}
