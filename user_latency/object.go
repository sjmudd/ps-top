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

// Object contains a table of rows
type Object struct {
	baseobject.BaseObject
	current Rows         // processlist
	results PlByUserRows // results by user
	totals  PlByUserRow  // totals of results
}

// NewUserLatency returns a user latency object
func NewUserLatency(ctx *context.Context) *Object {
	logger.Println("NewUserLatency()")
	o := new(Object)
	o.SetContext(ctx)

	return o
}

// Collect collects data from the db, updating initial
// values if needed, and then subtracting initial values if we want
// relative values, after which it stores totals.
func (t *Object) Collect(dbh *sql.DB) {
	logger.Println("Object.Collect() - starting collection of data")
	start := time.Now()

	t.current = selectRows(dbh)
	logger.Println("t.current collected", len(t.current), "row(s) from SELECT")

	t.processlist2byUser()

	logger.Println("Object.Collect() END, took:", time.Duration(time.Since(start)).String())
}

// Headings returns a string representing the view headings
func (t Object) Headings() string {
	return t.results.Headings()
}

// EmptyRowContent returns an empty string representing the view values
func (t Object) EmptyRowContent() string {
	return t.results.emptyRowContent()
}

// TotalRowContent returns a string representing the total view values
func (t Object) TotalRowContent() string {
	return t.totals.rowContent(t.totals)
}

// RowContent returns a string representing the row's view values
func (t Object) RowContent() []string {
	rows := make([]string, 0, len(t.results))

	for i := range t.results {
		rows = append(rows, t.results[i].rowContent(t.totals))
	}

	return rows
}

// Description returns a string description of the data being returned
func (t Object) Description() string {
	count := t.countRow()
	return fmt.Sprintf("Activity by Username (processlist) %d rows", count)
}

func (t Object) countRow() int {
	var count int
	for row := range t.results {
		if t.results[row].username != "" {
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
func (t *Object) processlist2byUser() {
	logger.Println("Object.processlist2byUser() START")

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

	for i := range t.current {
		// munge the username for special purposes (event scheduler, replication threads etc)
		id := t.current[i].ID
		username := t.current[i].user // limit size for display
		host := getHostname(t.current[i].host)
		command := t.current[i].command
		db := t.current[i].db
		info := t.current[i].info
		state := t.current[i].state

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
			row.username = t.current[i].user
			rowByUser[username] = row
		}
		row.connections++
		// ignore system SQL threads (may be more to filter out)
		if username != "system user" && host != "" && command != "Binlog Dump" {
			if command == "Sleep" {
				row.sleeptime += t.current[i].time
			} else {
				row.runtime += t.current[i].time
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
	t.results = results
	t.results.Sort() // sort output

	t.totals = t.results.totals()

	t.totals.hosts = uint64(len(globalHosts))
	t.totals.dbs = uint64(len(globalDbs))

	logger.Println("Object.processlist2byUser() END")
}

// Len returns the length of the result set
func (t Object) Len() int {
	return len(t.results)
}

func (t Object) HaveRelativeStats() bool {
	return false
}

// SetInitialFromCurrent - NOT IMPLEMENTED
func (t *Object) SetInitialFromCurrent() {
	logger.Println("user_latency.Object.SetInitialFromCurrent() NOT IMPLEMENTED")
}
