// Package userlatency manages the output from INFORMATION_SCHEMA.PROCESSLIST
package userlatency

// Rows contains a slice of Row rows
type Rows []Row

// totals returns the totals of all rows
func totals(rows Rows) Row {
	total := Row{Username: "Totals"}

	for _, row := range rows {
		total.Runtime += row.Runtime
		total.Sleeptime += row.Sleeptime
		total.Connections += row.Connections
		total.Active += row.Active
		total.Selects += row.Selects
		total.Inserts += row.Inserts
		total.Updates += row.Updates
		total.Deletes += row.Deletes
		total.Other += row.Other
	}

	return total
}
