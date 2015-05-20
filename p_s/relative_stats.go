package p_s

// RelativeStats records whether we want absolute counters as collected directly from the
// performance_schema tables or relative stats since when ps-top started.
type RelativeStats struct {
	wantRelativeStats bool
}

// SetWantRelativeStats records whether we want relative stats or not
func (wrs *RelativeStats) SetWantRelativeStats(wantRelativeStats bool) {
	wrs.wantRelativeStats = wantRelativeStats
}

// WantRelativeStats returns whether we want relative stats or not
func (wrs RelativeStats) WantRelativeStats() bool {
	return wrs.wantRelativeStats
}
