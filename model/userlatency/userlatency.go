// Package userlatency contains library routines for ps-top related to the INFORMATION_SCHEMA.PROCESSLIST table.
package userlatency

import (
	"database/sql"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/sjmudd/ps-top/config"
)

type mapStringInt map[string]int

// UserLatency contains a table of rows
type UserLatency struct {
	config         *config.Config
	FirstCollected time.Time
	LastCollected  time.Time
	current        []ProcesslistRow // processlist
	Results        []Row            // results by user
	Totals         Row              // totals of results
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

	ul.current = collect(ul.db)
	log.Println("t.current collected", len(ul.current), "row(s) from SELECT")

	ul.processlist2byUser()

	log.Println("UserLatency.Collect() END, took:", time.Duration(time.Since(start)).String())
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
	log.Println("UserLatency.processlist2byUser() START")

	reActiveReplMasterThread := regexp.MustCompile("Sending binlog event to slave")
	reSelect := regexp.MustCompile(`(?i)SELECT`) // make case insensitive
	reInsert := regexp.MustCompile(`(?i)INSERT`) // make case insensitive
	reUpdate := regexp.MustCompile(`(?i)UPDATE`) // make case insensitive
	reDelete := regexp.MustCompile(`(?i)DELETE`) // make case insensitive

	var (
		row     Row
		myHosts mapStringInt
		myDB    mapStringInt
		ok      bool
	)

	rowByUser := make(map[string]Row)
	hostsByUser := make(map[string]mapStringInt)
	dbsByUser := make(map[string]mapStringInt)

	// global values for totals.
	globalHosts := make(mapStringInt)
	globalDbs := make(mapStringInt)

	for i := range ul.current {
		// munge the Username for special purposes (event scheduler, replication threads etc)
		id := ul.current[i].ID
		Username := ul.current[i].user // limit size for display
		host := getHostname(ul.current[i].host)
		command := ul.current[i].command
		db := ul.current[i].db
		info := ul.current[i].info
		state := ul.current[i].state

		log.Println("- id/user/host:", id, Username, host)

		// fill global values
		if host != "" {
			globalHosts[host] = 1
		}
		if db != "" {
			globalDbs[db] = 1
		}

		if oldRow, ok := rowByUser[Username]; ok {
			log.Println("- found old row in rowByUser")
			row = oldRow // get old row
		} else {
			log.Println("- NOT found old row in rowByUser")
			// create new row - RESET THE VALUES !!!!
			rowp := new(Row)
			row = *rowp
			row.Username = ul.current[i].user
			rowByUser[Username] = row
		}
		row.Connections++
		// ignore system SQL threads (may be more to filter out)
		if Username != "system user" && host != "" && command != "Binlog Dump" {
			if command == "Sleep" {
				row.Sleeptime += ul.current[i].time
			} else {
				row.Runtime += ul.current[i].time
				row.Active++
			}
		}
		if command == "Binlog Dump" && reActiveReplMasterThread.MatchString(state) {
			row.Active++
		}

		// add the host if not known already
		if host != "" {
			if myHosts, ok = hostsByUser[Username]; !ok {
				myHosts = make(mapStringInt)
			}
			myHosts[host] = 1 // whatever - value doesn't matter
			hostsByUser[Username] = myHosts
		}
		row.Hosts = uint64(len(hostsByUser[Username]))

		// add the db count if not known already
		if db != "" {
			if myDB, ok = dbsByUser[Username]; !ok {
				myDB = make(mapStringInt)
			}
			myDB[db] = 1 // whatever - value doesn't matter
			dbsByUser[Username] = myDB
		}
		row.Dbs = uint64(len(dbsByUser[Username]))

		if reSelect.MatchString(info) {
			row.Selects++
		}
		if reInsert.MatchString(info) {
			row.Inserts++
		}
		if reUpdate.MatchString(info) {
			row.Updates++
		}
		if reDelete.MatchString(info) {
			row.Deletes++
		}

		rowByUser[Username] = row
	}

	results := make([]Row, 0, len(rowByUser))
	for _, v := range rowByUser {
		results = append(results, v)
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
