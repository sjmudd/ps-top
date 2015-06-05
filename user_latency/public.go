// Package user_latency contains library routines for ps-top related to the INFORMATION_SCHEMA.PROCESSLIST table.
package user_latency

import (
	"database/sql"
	"fmt"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/p_s"
	"regexp"
	"strings"
	"time"
)

type mapStringInt map[string]int

// Object contains a table of rows
type Object struct {
	p_s.RelativeStats
	p_s.CollectionTime
	current Rows      // processlist
	results PlByUserRows // results by user
	totals  PlByUserRow  // totals of results
}

// Collect collects data from the db, updating initial
// values if needed, and then subtracting initial values if we want
// relative values, after which it stores totals.
func (t *Object) Collect(dbh *sql.DB) {
	lib.Logger.Println("Object.Collect() - starting collection of data")
	start := time.Now()

	t.current = selectRows(dbh)
	lib.Logger.Println("t.current collected", len(t.current), "row(s) from SELECT")

	t.processlist2byUser()

	t.results.Sort()
	// lib.Logger.Println( "- collecting t.totals from t.results" )
	t.totals = t.results.totals()

	lib.Logger.Println("Object.Collect() END, took:", time.Duration(time.Since(start)).String())
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
func (t Object) RowContent(maxRows int) []string {
	rows := make([]string, 0, maxRows)

	for i := range t.results {
		if i < maxRows {
			rows = append(rows, t.results[i].rowContent(t.totals))
		}
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
	lib.Logger.Println("Object.processlist2byUser() START")

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

	for i := range t.current {
		// munge the username for special purposes (event scheduler, replication threads etc)
		id := t.current[i].ID
		username := t.current[i].user // limit size for display
		host := getHostname(t.current[i].host)
		command := t.current[i].command
		db := t.current[i].db
		info := t.current[i].info
		state := t.current[i].state

		lib.Logger.Println("- id/user/host:", id, username, host)

		if oldRow, ok := rowByUser[username]; ok {
			lib.Logger.Println("- found old row in rowByUser")
			row = oldRow // get old row
		} else {
			lib.Logger.Println("- NOT found old row in rowByUser")
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

	lib.Logger.Println("Object.processlist2byUser() END")
}

// Len returns the length of the result set
func (t Object) Len() int {
	return len(t.results)
}
