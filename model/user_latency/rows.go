// Package user_latency manages the output from INFORMATION_SCHEMA.PROCESSLIST
package user_latency

import (
	"sort"
)

// Rows contains a slice of Row rows
type Rows []Row

// generate a row of totals from a table
func (t Rows) totals() Row {
	var totals Row
	totals.Username = "Totals"

	for i := range t {
		totals.Runtime += t[i].Runtime
		totals.Sleeptime += t[i].Sleeptime
		totals.Connections += t[i].Connections
		totals.Active += t[i].Active
		totals.Selects += t[i].Selects
		totals.Inserts += t[i].Inserts
		totals.Updates += t[i].Updates
		totals.Deletes += t[i].Deletes
		totals.Other += t[i].Other
	}

	return totals
}

// ByRunTime is for sorting rows by Runtime
type ByRunTime Rows

func (t ByRunTime) Len() int      { return len(t) }
func (t ByRunTime) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t ByRunTime) Less(i, j int) bool {
	return (t[i].TotalTime() > t[j].TotalTime()) ||
		((t[i].TotalTime() == t[j].TotalTime()) && (t[i].Connections > t[j].Connections)) ||
		((t[i].TotalTime() == t[j].TotalTime()) && (t[i].Connections == t[j].Connections) && (t[i].Username < t[j].Username))
}

// Sort by User rows
func (t Rows) Sort() {
	sort.Sort(ByRunTime(t))
}
