// i_s - library routines for pstop.
//
// This file contains the library routines for managing the
// table_io_waits_by_table table.
package processlist

import (
	"database/sql"
	"fmt"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/p_s"
	"regexp"
	"strings"
	"time"
)

type map_string_int map[string]int

// a table of rows
type Object struct {
	p_s.RelativeStats
	p_s.CollectionTime
	current table_rows      // processlist
	results pl_by_user_rows // results by user
	totals  pl_by_user_row  // totals of results
}

// Collect() collects data from the db, updating initial
// values if needed, and then subtracting initial values if we want
// relative values, after which it stores totals.
func (t *Object) Collect(dbh *sql.DB) {
	lib.Logger.Println("Object.Collect() - starting collection of data")
	start := time.Now()

	t.current = select_processlist(dbh)
	lib.Logger.Println("t.current collected", len(t.current), "row(s) from SELECT")

	t.processlist2by_user()

	t.results.Sort()
	// lib.Logger.Println( "- collecting t.totals from t.results" )
	t.totals = t.results.totals()

	lib.Logger.Println("Object.Collect() END, took:", time.Duration(time.Since(start)).String())
}

func (t Object) Headings() string {
	return t.results.Headings()
}

func (t Object) EmptyRowContent() string {
	return t.results.emptyRowContent()
}

func (t Object) TotalRowContent() string {
	return t.totals.row_content(t.totals)
}

func (t Object) RowContent(max_rows int) []string {
	rows := make([]string, 0, max_rows)

	for i := range t.results {
		if i < max_rows {
			rows = append(rows, t.results[i].row_content(t.totals))
		}
	}

	return rows
}

func (t Object) Description() string {
	count := t.count_rows()
	return fmt.Sprintf("Activity by Username (processlist) %d rows", count)
}

func (t Object) count_rows() int {
	var count int
	for row := range t.results {
		if t.results[row].username != "" {
			count++
		}
	}
	return count
}

// return the hostname without the port part
func get_hostname(h_p string) string {
	i := strings.Index(h_p, ":")
	if i >= 0 {
		return h_p[0:i]
	} else {
		return h_p // shouldn't happen !!!
	}
}

// read in processlist and add the appropriate values into a new pl_by_user table
func (t *Object) processlist2by_user() {
	lib.Logger.Println("Object.processlist2by_user() START")

	var re_active_repl_master_thread *regexp.Regexp = regexp.MustCompile("Sending binlog event to slave")
	var re_select *regexp.Regexp = regexp.MustCompile(`(?i)SELECT`) // make case insensitive
	var re_insert *regexp.Regexp = regexp.MustCompile(`(?i)INSERT`) // make case insensitive
	var re_update *regexp.Regexp = regexp.MustCompile(`(?i)UPDATE`) // make case insensitive
	var re_delete *regexp.Regexp = regexp.MustCompile(`(?i)DELETE`) // make case insensitive

	var row pl_by_user_row
	var results pl_by_user_rows
	var my_hosts map_string_int
	var my_db map_string_int
	var ok bool

	row_by_user := make(map[string]pl_by_user_row)
	hosts_by_user := make(map[string]map_string_int)
	dbs_by_user := make(map[string]map_string_int)

	for i := range t.current {
		// munge the username for special purposes (event scheduler, replication threads etc)
		id := t.current[i].ID
		username := t.current[i].USER // limit size for display
		host := get_hostname(t.current[i].HOST)
		command := t.current[i].COMMAND
		db := t.current[i].DB
		info := t.current[i].INFO
		state := t.current[i].STATE

		lib.Logger.Println("- id/user/host:", id, username, host)

		if old_row, ok := row_by_user[username]; ok {
			lib.Logger.Println("- found old row in row_by_user")
			row = old_row // get old row
		} else {
			lib.Logger.Println("- NOT found old row in row_by_user")
			// create new row - RESET THE VALUES !!!!
			rowp := new(pl_by_user_row)
			row = *rowp
			row.username = t.current[i].USER
			row_by_user[username] = row
		}
		row.connections++
		// ignore system SQL threads (may be more to filter out)
		if username != "system user" && host != "" && command != "Binlog Dump" {
			if command == "Sleep" {
				row.sleeptime += t.current[i].TIME
			} else {
				row.runtime += t.current[i].TIME
				row.active++
			}
		}
		if command == "Binlog Dump" && re_active_repl_master_thread.MatchString(state) {
			row.active++
		}

		// add the host if not known already
		if host != "" {
			if my_hosts, ok = hosts_by_user[username]; !ok {
				my_hosts = make(map_string_int)
			}
			my_hosts[host] = 1 // whatever - value doesn't matter
			hosts_by_user[username] = my_hosts
		}
		row.hosts = uint64(len(hosts_by_user[username]))

		// add the db count if not known already
		if db != "" {
			if my_db, ok = dbs_by_user[username]; !ok {
				my_db = make(map_string_int)
			}
			my_db[db] = 1 // whatever - value doesn't matter
			dbs_by_user[username] = my_db
		}
		row.dbs = uint64(len(dbs_by_user[username]))

		if re_select.MatchString(info) == true {
			row.selects++
		}
		if re_insert.MatchString(info) == true {
			row.inserts++
		}
		if re_update.MatchString(info) == true {
			row.updates++
		}
		if re_delete.MatchString(info) == true {
			row.deletes++
		}

		row_by_user[username] = row
	}

	results = make(pl_by_user_rows, 0, len(row_by_user))
	for _, v := range row_by_user {
		results = append(results, v)
	}
	t.results = results
	t.results.Sort() // sort output

	t.totals = t.results.totals()

	lib.Logger.Println("Object.processlist2by_user() END")
}

// return the length of the result set
func (t Object) Len() int {
	return len(t.results)
}
