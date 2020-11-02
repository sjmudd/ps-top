// Package user_latency contains library routines for ps-top related to the INFORMATION_SCHEMA.PROCESSLIST table.
package user_latency

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/sjmudd/ps-top/baseobject"
	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/logger"
)

type mapStringInt map[string]int

// UserLatency contains a table of rows
type UserLatency struct {
	baseobject.BaseObject
	current Rows         // processlist
	results PlByUserRows // results by user
	totals  PlByUserRow  // totals of results
	db      *sql.DB
}

// NewUserLatency returns a user latency object
func NewUserLatency(ctx *context.Context, db *sql.DB) *UserLatency {
	logger.Println("NewUserLatency()")
	ul := &UserLatency{
		db: db,
	}
	ul.SetContext(ctx)

	return ul
}

// Collect collects data from the db, updating initial
// values if needed, and then subtracting initial values if we want
// relative values, after which it stores totals.
func (ul *UserLatency) Collect() {
	logger.Println("UserLatency.Collect() - starting collection of data")
	start := time.Now()

	ul.current = collect(ul.db)
	logger.Println("t.current collected", len(ul.current), "row(s) from SELECT")

	ul.processlist2byUser()

	logger.Println("UserLatency.Collect() END, took:", time.Duration(time.Since(start)).String())
}

// Headings returns a string representing the view headings
func (ul UserLatency) Headings() string {
	return ul.results.Headings()
}

// EmptyRowContent returns an empty string representing the view values
func (ul UserLatency) EmptyRowContent() string {
	return ul.results.emptyRowContent()
}

// TotalRowContent returns a string representing the total view values
func (ul UserLatency) TotalRowContent() string {
	return ul.totals.content(ul.totals)
}

// RowContent returns a string representing the row's view values
func (ul UserLatency) RowContent() []string {
	rows := make([]string, 0, len(ul.results))

	for i := range ul.results {
		rows = append(rows, ul.results[i].content(ul.totals))
	}

	return rows
}

// Description returns a string description of the data being returned
func (ul UserLatency) Description() string {
	count := ul.countRow()
	return fmt.Sprintf("Activity by Username (processlist) %d rows", count)
}

func (ul UserLatency) countRow() int {
	var count int
	for row := range ul.results {
		if ul.results[row].username != "" {
			count++
		}
	}
	return count
}

// return the hostname without the port part
func getHostname(hostPort string) string {
	i := strings.Index(hostPort, ":")
	if i >= 0 {
		return hostPort[0:i]
	}
	return hostPort // shouldn't happen !!!
}

// read in processlist and add the appropriate values into a new pl_by_user table
func (ul *UserLatency) processlist2byUser() {
	logger.Println("UserLatency.processlist2byUser() START")

	reActiveReplMasterThread := regexp.MustCompile("Sending binlog event to slave")
	reSelect := regexp.MustCompile(`(?i)SELECT`) // make case insensitive
	reInsert := regexp.MustCompile(`(?i)INSERT`) // make case insensitive
	reUpdate := regexp.MustCompile(`(?i)UPDATE`) // make case insensitive
	reDelete := regexp.MustCompile(`(?i)DELETE`) // make case insensitive

	var row PlByUserRow
	var results PlByUserRows
	var myHosts mapStringInt
	var myDB mapStringInt
	var ok bool

	rowByUser := make(map[string]PlByUserRow)
	hostsByUser := make(map[string]mapStringInt)
	DBsByUser := make(map[string]mapStringInt)

	// global values for totals.
	globalHosts := make(map[string]int)
	globalDbs := make(map[string]int)

	for i := range ul.current {
		// munge the username for special purposes (event scheduler, replication threads etc)
		id := ul.current[i].ID
		username := ul.current[i].user // limit size for display
		host := getHostname(ul.current[i].host)
		command := ul.current[i].command
		db := ul.current[i].db
		info := ul.current[i].info
		state := ul.current[i].state

		logger.Println("- id/user/host:", id, username, host)

		// fill global values
		if host != "" {
			globalHosts[host] = 1
		}
		if db != "" {
			globalDbs[db] = 1
		}

		if oldRow, ok := rowByUser[username]; ok {
			logger.Println("- found old row in rowByUser")
			row = oldRow // get old row
		} else {
			logger.Println("- NOT found old row in rowByUser")
			// create new row - RESET THE VALUES !!!!
			rowp := new(PlByUserRow)
			row = *rowp
			row.username = ul.current[i].user
			rowByUser[username] = row
		}
		row.connections++
		// ignore system SQL threads (may be more to filter out)
		if username != "system user" && host != "" && command != "Binlog Dump" {
			if command == "Sleep" {
				row.sleeptime += ul.current[i].time
			} else {
				row.runtime += ul.current[i].time
				row.active++
			}
		}
		if command == "Binlog Dump" && reActiveReplMasterThread.MatchString(state) {
			row.active++
		}

		// add the host if not known already
		if host != "" {
			if myHosts, ok = hostsByUser[username]; !ok {
				myHosts = make(mapStringInt)
			}
			myHosts[host] = 1 // whatever - value doesn't matter
			hostsByUser[username] = myHosts
		}
		row.hosts = uint64(len(hostsByUser[username]))

		// add the db count if not known already
		if db != "" {
			if myDB, ok = DBsByUser[username]; !ok {
				myDB = make(mapStringInt)
			}
			myDB[db] = 1 // whatever - value doesn't matter
			DBsByUser[username] = myDB
		}
		row.dbs = uint64(len(DBsByUser[username]))

		if reSelect.MatchString(info) == true {
			row.selects++
		}
		if reInsert.MatchString(info) == true {
			row.inserts++
		}
		if reUpdate.MatchString(info) == true {
			row.updates++
		}
		if reDelete.MatchString(info) == true {
			row.deletes++
		}

		rowByUser[username] = row
	}

	results = make(PlByUserRows, 0, len(rowByUser))
	for _, v := range rowByUser {
		results = append(results, v)
	}
	ul.results = results
	ul.results.Sort() // sort output

	ul.totals = ul.results.totals()

	ul.totals.hosts = uint64(len(globalHosts))
	ul.totals.dbs = uint64(len(globalDbs))

	logger.Println("UserLatency.processlist2byUser() END")
}

// Len returns the length of the result set
func (ul UserLatency) Len() int {
	return len(ul.results)
}

func (ul UserLatency) HaveRelativeStats() bool {
	return false
}

// SetInitialFromCurrent - NOT IMPLEMENTED
func (ul *UserLatency) SetInitialFromCurrent() {
	logger.Println("user_latency.UserLatency.SetInitialFromCurrent() NOT IMPLEMENTED")
}
