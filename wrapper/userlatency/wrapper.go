// Package userlatency holds the routines which manage the user latency information
package userlatency

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/userlatency"
	"github.com/sjmudd/ps-top/utils"
	"github.com/sjmudd/ps-top/wrapper"
)

// Wrapper wraps a UserLatency struct
type Wrapper struct {
	ul *userlatency.UserLatency
}

// NewUserLatency creates a wrapper around UserLatency
func NewUserLatency(cfg *config.Config, db *sql.DB) *Wrapper {
	return &Wrapper{
		ul: userlatency.NewUserLatency(cfg, db),
	}
}

// ResetStatistics resets the statistics to last values
func (ulw *Wrapper) ResetStatistics() {
	ulw.ul.ResetStatistics()
}

// Collect data from the db, then sort the results.
func (ulw *Wrapper) Collect() {
	ulw.ul.Collect()
	sort.Sort(byTotalTime(ulw.ul.Results))
}

// RowContent returns the rows we need for displaying
func (ulw Wrapper) RowContent() []string {
	n := len(ulw.ul.Results)
	return wrapper.RowsFromGetter(n, func(i int) string {
		return ulw.content(ulw.ul.Results[i], ulw.ul.Totals)
	})
}

// TotalRowContent returns all the totals
func (ulw Wrapper) TotalRowContent() string {
	return wrapper.TotalRowContent(ulw.ul.Totals, ulw.content)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (ulw Wrapper) EmptyRowContent() string {
	return wrapper.EmptyRowContent(ulw.content)
}

// HaveRelativeStats is true for this object
func (ulw Wrapper) HaveRelativeStats() bool {
	return ulw.ul.HaveRelativeStats()
}

// FirstCollectTime returns the time the first value was collected
func (ulw Wrapper) FirstCollectTime() time.Time {
	return ulw.ul.FirstCollected
}

// LastCollectTime returns the time the last value was collected
func (ulw Wrapper) LastCollectTime() time.Time {
	return ulw.ul.LastCollected
}

// WantRelativeStats indicates if we want relative statistics
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
	return fmt.Sprintf("%-10s %6s|%-10s %6s|%4s %4s|%5s %3s|%3s %3s %3s %3s %3s|%s",
		"Run Time", "%", "Sleeping", "%", "Conn", "Actv", "Hosts", "DBs", "Sel", "Ins", "Upd", "Del", "Oth", "User")
}

// content generate a printable result for a row, given the totals
func (ulw Wrapper) content(row, totals userlatency.Row) string {
	return fmt.Sprintf("%10s %6s|%10s %6s|%4s %4s|%5s %3s|%3s %3s %3s %3s %3s|%s",
		formatSeconds(row.Runtime),
		utils.FormatPct(utils.Divide(row.Runtime, totals.Runtime)),
		formatSeconds(row.Sleeptime),
		utils.FormatPct(utils.Divide(row.Sleeptime, totals.Sleeptime)),
		utils.FormatCounterU(row.Connections, 4),
		utils.FormatCounterU(row.Active, 4),
		utils.FormatCounterU(row.Hosts, 5),
		utils.FormatCounterU(row.Dbs, 3),
		utils.FormatCounterU(row.Selects, 3),
		utils.FormatCounterU(row.Inserts, 3),
		utils.FormatCounterU(row.Updates, 3),
		utils.FormatCounterU(row.Deletes, 3),
		utils.FormatCounterU(row.Other, 3),
		row.Username)
}

// byTotalTime is for sorting rows by Runtime
type byTotalTime []userlatency.Row

func (t byTotalTime) Len() int      { return len(t) }
func (t byTotalTime) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t byTotalTime) Less(i, j int) bool {
	return (t[i].TotalTime() > t[j].TotalTime()) ||
		((t[i].TotalTime() == t[j].TotalTime()) && (t[i].Connections > t[j].Connections)) ||
		((t[i].TotalTime() == t[j].TotalTime()) && (t[i].Connections == t[j].Connections) && (t[i].Username < t[j].Username))
}

// formatSeconds formats the given seconds into xxh xxm xxs or xxd xxh xxm
// for periods longer than 24h.  If seconds is 0 return an empty string.
// Leading 0 values are omitted.
// e.g.  0  -> ""
//
//	   10 -> "10s"
//	   70 -> "1m 10s"
//	 3601 -> "1h 0m 1s"
//	86400 -> "1d 0h 0m"
//
// Note: we assume a 10 character width as formatting will get messed up so if there's not enough space don't add the lower values.
func formatSeconds(d uint64) string {
	if d == 0 {
		return ""
	}

	days := d / 86400
	hours := (d - days*86400) / 3600
	minutes := (d - days*86400 - hours*3600) / 60
	seconds := d - days*86400 - hours*3600 - minutes*60

	if days > 0 {
		result := fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
		if len(result) > 10 {
			result = fmt.Sprintf("%dd %dh", days, hours)
		}
		return result
	}
	if hours > 0 {
		result := fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
		if len(result) > 10 {
			result = fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return result
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}

	return fmt.Sprintf("%ds", seconds)
}
