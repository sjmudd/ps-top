// Package userlatency contains library routines for ps-top related to the INFORMATION_SCHEMA.PROCESSLIST table.
package userlatency

import (
	"database/sql"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/processlist"
)

type mapStringInt map[string]int

// UserLatency contains a table of rows
type UserLatency struct {
	config         *config.Config
	FirstCollected time.Time
	LastCollected  time.Time
	current        []processlist.Row // processlist
	Results        []Row             // results by user
	Totals         Row               // totals of results
	db             *sql.DB
}

// NewUserLatency returns a user latency object
func NewUserLatency(cfg *config.Config, db *sql.DB) *UserLatency {
	log.Println("NewUserLatency()")
	ul := &UserLatency{
		config: cfg,
		db:     db,
	}

	return ul
}

// Collect collects data from the db, updating initial
// values if needed, and then subtracting initial values if we want
// relative values, after which it stores totals.
func (ul *UserLatency) Collect() {
	log.Println("UserLatency.Collect() - starting collection of data")
	start := time.Now()

	ul.current = processlist.Collect(ul.db)
	log.Println("t.current collected", len(ul.current), "row(s) from SELECT")

	ul.processlist2byUser()

	log.Println("UserLatency.Collect() END, took:", time.Since(start).String())
}

// return the hostname without the port part
func getHostname(hostPort string) string {
	i := strings.Index(hostPort, ":")
	if i >= 0 {
		return hostPort[0:i]
	}
	return hostPort // shouldn't happen !!!
}

// helper: get or create a Row pointer for username
func getOrCreateRow(rowByUser map[string]*Row, username, origUser string) *Row {
	if r, ok := rowByUser[username]; ok {
		return r
	}
	r := &Row{Username: origUser}
	rowByUser[username] = r
	return r
}

// helper: update runtime and active counters
func updateRuntimeAndActive(r *Row, command string, t uint64, host, state string, reActive *regexp.Regexp) {
	if r.Username != "system user" && host != "" && command != "Binlog Dump" {
		if command == "Sleep" {
			r.Sleeptime += t
		} else {
			r.Runtime += t
			r.Active++
		}
	}
	if command == "Binlog Dump" && reActive.MatchString(state) {
		r.Active++
	}
}

// helper: add host to hostsByUser and return count of distinct hosts for user
func addHost(hostsByUser map[string]mapStringInt, username, host string) uint64 {
	if host == "" {
		return 0
	}
	myHosts, ok := hostsByUser[username]
	if !ok {
		myHosts = make(mapStringInt)
	}
	myHosts[host] = 1
	hostsByUser[username] = myHosts
	return uint64(len(myHosts))
}

// helper: add db to dbsByUser and return count of distinct dbs for user
func addDB(dbsByUser map[string]mapStringInt, username, db string) uint64 {
	if db == "" {
		return 0
	}
	myDB, ok := dbsByUser[username]
	if !ok {
		myDB = make(mapStringInt)
	}
	myDB[db] = 1
	dbsByUser[username] = myDB
	return uint64(len(myDB))
}

// helper: increment statement counters based on info
func addStatementCounts(r *Row, info string, reSelect, reInsert, reUpdate, reDelete *regexp.Regexp) {
	if reSelect.MatchString(info) {
		r.Selects++
	}
	if reInsert.MatchString(info) {
		r.Inserts++
	}
	if reUpdate.MatchString(info) {
		r.Updates++
	}
	if reDelete.MatchString(info) {
		r.Deletes++
	}
}

// read in processlist and add the appropriate values into a new pl_by_user table
func (ul *UserLatency) processlist2byUser() {
	log.Println("UserLatency.processlist2byUser() START")

	reActiveReplMasterThread := regexp.MustCompile("Sending binlog event to slave")
	reSelect := regexp.MustCompile(`(?i)SELECT`) // make case insensitive
	reInsert := regexp.MustCompile(`(?i)INSERT`) // make case insensitive
	reUpdate := regexp.MustCompile(`(?i)UPDATE`) // make case insensitive
	reDelete := regexp.MustCompile(`(?i)DELETE`) // make case insensitive

	rowByUser := make(map[string]*Row)
	hostsByUser := make(map[string]mapStringInt)
	dbsByUser := make(map[string]mapStringInt)

	// global values for totals.
	globalHosts := make(mapStringInt)
	globalDbs := make(mapStringInt)

	for i := range ul.current {
		pl := ul.current[i]
		id := pl.ID
		Username := pl.User // limit size for display
		host := getHostname(pl.Host)
		command := pl.Command
		db := pl.Db
		info := pl.Info
		state := pl.State

		log.Println("- id/user/host:", id, Username, host)

		// fill global values
		if host != "" {
			globalHosts[host] = 1
		}
		if db != "" {
			globalDbs[db] = 1
		}

		r := getOrCreateRow(rowByUser, Username, pl.User)
		log.Println("- processing row for user:", Username)

		r.Connections++

		updateRuntimeAndActive(r, command, pl.Time, host, state, reActiveReplMasterThread)

		// track hosts and dbs per user
		r.Hosts = addHost(hostsByUser, Username, host)
		r.Dbs = addDB(dbsByUser, Username, db)

		addStatementCounts(r, info, reSelect, reInsert, reUpdate, reDelete)
	}

	results := make([]Row, 0, len(rowByUser))
	for _, v := range rowByUser {
		results = append(results, *v)
	}
	ul.Results = results
	ul.Totals = totals(ul.Results)
	ul.Totals.Hosts = uint64(len(globalHosts))
	ul.Totals.Dbs = uint64(len(globalDbs))

	log.Println("UserLatency.processlist2byUser() END")
}

// HaveRelativeStats returns if we have relative information
func (ul UserLatency) HaveRelativeStats() bool {
	return false
}

// WantRelativeStats
func (ul UserLatency) WantRelativeStats() bool {
	return ul.config.WantRelativeStats()
}

// ResetStatistics - NOT IMPLEMENTED
func (ul *UserLatency) ResetStatistics() {
	log.Println("userlatency.UserLatency.ResetStatistics() NOT IMPLEMENTED")
}
