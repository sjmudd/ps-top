// performance_schema - library routines for pstop.
//
// want_relative_stats
package performance_schema

// a table of rows
type RelativeStats struct {
	want_relative_stats bool
}

func (wrs *RelativeStats) SetWantRelativeStats(want_relative_stats bool) {
	wrs.want_relative_stats = want_relative_stats
}

func (wrs RelativeStats) WantRelativeStats() bool {
	return wrs.want_relative_stats
}
