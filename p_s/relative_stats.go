// performance_schema - library routines for pstop.
//
// want_relative_stats
package p_s

// a table of rows
type RelativeStats struct {
	want_relative_stats bool
}

// set if we want relative stats
func (wrs *RelativeStats) SetWantRelativeStats(want_relative_stats bool) {
	wrs.want_relative_stats = want_relative_stats
}

// return if we want relative stats
func (wrs RelativeStats) WantRelativeStats() bool {
	return wrs.want_relative_stats
}
