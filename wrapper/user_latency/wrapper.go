// Package user_latency holds the routines which manage the user latency information
package user_latency

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/model/user_latency"
)

// Wrapper wraps a UserLatency struct
type Wrapper struct {
	ul *user_latency.UserLatency
}

// NewUserLatency creates a wrapper around UserLatency
func NewUserLatency(ctx *context.Context, db *sql.DB) *Wrapper {
	return &Wrapper{
		ul: user_latency.NewUserLatency(ctx, db),
	}
}

// SetFirstFromLast resets the statistics to last values
func (ulw *Wrapper) SetFirstFromLast() {
	ulw.ul.SetFirstFromLast()
}

// Collect data from the db, then merge it in.
func (ulw *Wrapper) Collect() {
	ulw.ul.Collect()
}

// RowContent returns the rows we need for displaying
func (ulw Wrapper) RowContent() []string {
	rows := make([]string, 0, len(ulw.ul.Results))

	for i := range ulw.ul.Results {
		rows = append(rows, ulw.content(ulw.ul.Results[i], ulw.ul.Totals))
	}

	return rows
}

// TotalRowContent returns all the totals
func (ulw Wrapper) TotalRowContent() string {
	return ulw.content(ulw.ul.Totals, ulw.ul.Totals)
}

// Len return the length of the result set
func (ulw Wrapper) Len() int {
	return len(ulw.ul.Results)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (ulw Wrapper) EmptyRowContent() string {
	var empty user_latency.Row

	return ulw.content(empty, empty)
}

// HaveRelativeStats is true for this object
func (ulw Wrapper) HaveRelativeStats() bool {
	return ulw.ul.HaveRelativeStats()
}

// FirstCollectTime returns the time the first value was collected
func (ulw Wrapper) FirstCollectTime() time.Time {
	return ulw.ul.FirstCollectTime()
}

// LastCollectTime returns the time the last value was collected
func (ulw Wrapper) LastCollectTime() time.Time {
	return ulw.ul.LastCollectTime()
}

// WantRelativeStats indiates if we want relative statistics
func (ulw Wrapper) WantRelativeStats() bool {
	return ulw.ul.WantRelativeStats()
}

// Description returns a description of the table
func (ulw Wrapper) Description() string {
	var count int
	for row := range ulw.ul.Results {
		if ulw.ul.Results[row].Username != "" {
			count++
		}
	}
	return fmt.Sprintf("Activity by Username (processlist) %d rows", count)
}

// Headings returns the headings for a table
func (ulw Wrapper) Headings() string {
	return fmt.Sprintf("%-8s %6s|%-8s %6s|%4s %4s|%5s %3s|%3s %3s %3s %3s %3s|%s",
		"Run Time", "%", "Sleeping", "%", "Conn", "Actv", "Hosts", "DBs", "Sel", "Ins", "Upd", "Del", "Oth", "User")
}

// content generate a printable result for a row, given the totals
func (ulw Wrapper) content(row, totals user_latency.Row) string {
	return fmt.Sprintf("%8s %6s|%8s %6s|%4s %4s|%5s %3s|%3s %3s %3s %3s %3s|%s",
		lib.FormatSeconds(row.Runtime),
		lib.FormatPct(lib.Divide(row.Runtime, totals.Runtime)),
		lib.FormatSeconds(row.Sleeptime),
		lib.FormatPct(lib.Divide(row.Sleeptime, totals.Sleeptime)),
		lib.FormatCounter(int(row.Connections), 4),
		lib.FormatCounter(int(row.Active), 4),
		lib.FormatCounter(int(row.Hosts), 5),
		lib.FormatCounter(int(row.Dbs), 3),
		lib.FormatCounter(int(row.Selects), 3),
		lib.FormatCounter(int(row.Inserts), 3),
		lib.FormatCounter(int(row.Updates), 3),
		lib.FormatCounter(int(row.Deletes), 3),
		lib.FormatCounter(int(row.Other), 3),
		row.Username)
}
