// Package userlatency holds the routines which manage the user latency information.
package userlatency

import (
	"database/sql"
	"fmt"
	"slices"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/userlatency"
	"github.com/sjmudd/ps-top/presenter"
	"github.com/sjmudd/ps-top/utils"
)

// formatSeconds formats the given seconds into xxh xxm xxs or xxd xxh xxm
// for periods longer than 24h. If seconds is 0 return an empty string.
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

var (
	defaultSort = func(rows []userlatency.Row) {
		slices.SortFunc(rows, func(a, b userlatency.Row) int {
			if a.TotalTime() > b.TotalTime() {
				return -1
			}
			if a.TotalTime() < b.TotalTime() {
				return 1
			}
			if a.Connections > b.Connections {
				return -1
			}
			if a.Connections < b.Connections {
				return 1
			}
			if a.Username < b.Username {
				return -1
			}
			if a.Username > b.Username {
				return 1
			}
			return 0
		})
	}

	defaultHasData = func(r userlatency.Row) bool { return r.Username != "" }

	defaultContent = func(row, totals userlatency.Row) string {
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
)

// Presenter presents a UserLatency struct.
type Presenter struct {
	*presenter.BasePresenter[userlatency.Row, *userlatency.UserLatency]
}

// NewUserLatency creates a presenter for UserLatency.
func NewUserLatency(cfg *config.Config, db *sql.DB) *Presenter {
	ul := userlatency.NewUserLatency(cfg, db)
	bp := presenter.NewBasePresenter(
		ul,
		"Activity by Username (processlist)",
		defaultSort,
		defaultHasData,
		defaultContent,
	)
	return &Presenter{BasePresenter: bp}
}

// Headings returns the headings for a table.
func (p *Presenter) Headings() string {
	return fmt.Sprintf("%-10s %6s|%-10s %6s|%4s %4s|%5s %3s|%3s %3s %3s %3s %3s|%s",
		"Run Time", "%", "Sleeping", "%", "Conn", "Actv", "Hosts", "DBs", "Sel", "Ins", "Upd", "Del", "Oth", "User")
}
