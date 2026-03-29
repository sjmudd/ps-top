// Package userlatency contains library routines for ps-top related to the INFORMATION_SCHEMA.PROCESSLIST table.
package userlatency

import (
	"regexp"
	"strings"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model"
	"github.com/sjmudd/ps-top/model/processlist"
)

type mapStringInt map[string]int

// UserLatency aggregates processlist data by user
type UserLatency struct {
	*model.BaseCollector[Row, []Row]
}

// NewUserLatency creates a new UserLatency instance.
func NewUserLatency(cfg *config.Config, db model.QueryExecutor) *UserLatency {
	process := func(last, _ []Row) ([]Row, Row) {
		// last already contains aggregated rows; just copy and compute totals
		results := make([]Row, len(last))
		copy(results, last)
		tot := totals(results)
		return results, tot
	}
	bc := model.NewBaseCollector[Row, []Row](cfg, db, process)
	return &UserLatency{BaseCollector: bc}
}

// Collect fetches processlist data, aggregates by user, and updates results.
func (ul *UserLatency) Collect() {
	bc := ul.BaseCollector
	fetch := func() ([]Row, error) {
		raw := processlist.Collect(bc.DB())
		aggregated := ul.processlist2byUser(raw)
		return aggregated, nil
	}
	wantRefresh := func() bool {
		// Refresh baseline on first collection only
		return len(bc.First) == 0 && len(bc.Last) > 0
	}
	bc.Collect(fetch, wantRefresh)
}

// processlist2byUser aggregates raw processlist rows by username
func (ul *UserLatency) processlist2byUser(raw []processlist.Row) []Row {
	reActiveReplMasterThread := regexp.MustCompile("Sending binlog event to slave")
	reSelect := regexp.MustCompile(`(?i)SELECT`)
	reInsert := regexp.MustCompile(`(?i)INSERT`)
	reUpdate := regexp.MustCompile(`(?i)UPDATE`)
	reDelete := regexp.MustCompile(`(?i)DELETE`)

	rowByUser := make(map[string]*Row)
	hostsByUser := make(map[string]mapStringInt)
	dbsByUser := make(map[string]mapStringInt)
	globalHosts := make(mapStringInt)
	globalDbs := make(mapStringInt)

	for i := range raw {
		pl := raw[i]
		username := pl.User
		host := getHostname(pl.Host)
		command := pl.Command
		db := pl.Db
		info := pl.Info
		state := pl.State

		// fill global values
		if host != "" {
			globalHosts[host] = 1
		}
		if db != "" {
			globalDbs[db] = 1
		}

		r := getOrCreateRow(rowByUser, username, pl.User)
		r.Connections++

		updateRuntimeAndActive(r, command, pl.Time, host, state, reActiveReplMasterThread)

		// track hosts and dbs per user
		r.Hosts = addHost(hostsByUser, username, host)
		r.Dbs = addDB(dbsByUser, username, db)

		addStatementCounts(r, info, reSelect, reInsert, reUpdate, reDelete)
	}

	results := make([]Row, 0, len(rowByUser))
	for _, v := range rowByUser {
		results = append(results, *v)
	}
	// Totals are computed later by BaseCollector's process function; not computed here.
	_ = totals(results) // ensure totals function exists; but not storing here
	return results
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

// HaveRelativeStats returns false for this model (no baseline subtraction)
func (ul UserLatency) HaveRelativeStats() bool {
	return false
}

// WantRelativeStats returns the config setting.
func (ul UserLatency) WantRelativeStats() bool {
	return ul.Config().WantRelativeStats()
}

// return the hostname without the port part
func getHostname(hostPort string) string {
	i := strings.Index(hostPort, ":")
	if i >= 0 {
		return hostPort[0:i]
	}
	return hostPort // shouldn't happen !!!!
}
