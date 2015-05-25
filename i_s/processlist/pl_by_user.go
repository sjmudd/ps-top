// Package processlist manages the output from INFORMATION_SCHEMA.PROCESSLIST
package processlist

import (
	"fmt"
	"sort"

	"github.com/sjmudd/ps-top/lib"
)

/*
root@localhost [i_s]> show create table i_s\G
*************************** 1. row ***************************
CREATE TEMPORARY TABLE `PROCESSLIST` (
	`ID` bigint(21) unsigned NOT NULL DEFAULT '0',
	`USER` varchar(16) NOT NULL DEFAULT '',
	`HOST` varchar(64) NOT NULL DEFAULT '',
	`DB` varchar(64) DEFAULT NULL, `COMMAND` varchar(16) NOT NULL DEFAULT '', `TIME` int(7) NOT NULL DEFAULT '0', `STATE` varchar(64) DEFAULT NULL,
	`INFO` longtext
) ENGINE=MyISAM DEFAULT CHARSET=utf8
1 row in set (0.02 sec)
*/

// PlByUserRow contains a summary row of information taken from information_schema.processlist
type PlByUserRow struct {
	username    string
	runtime     uint64
	sleeptime   uint64
	connections uint64
	active      uint64
	hosts       uint64
	dbs         uint64
	selects     uint64
	inserts     uint64
	updates     uint64
	deletes     uint64
	other       uint64
}
// PlByUserRows contains a slice of PlByUserRow rows
type PlByUserRows []PlByUserRow

/*
Run Time   %age|Sleeping      %|Conn Actv|Hosts DBs|Sel Ins Upd Del Oth|username
hh:mm:ss 100.0%|hh:mm:ss 100.0%|9999 9999|9999  999|999 999 999 999 999|xxxxxxxxxxxxxx
*/

func (r *PlByUserRow) headings() string {
	return fmt.Sprintf("%-8s %6s|%-8s %6s|%4s %4s|%5s %3s|%3s %3s %3s %3s %3s|%s",
		"Run Time", "%", "Sleeping", "%", "Conn", "Actv", "Hosts", "DBs", "Sel", "Ins", "Upd", "Del", "Oth", "User")
}

// generate a printable result
func (r *PlByUserRow) rowContent(totals PlByUserRow) string {
	return fmt.Sprintf("%8s %6s|%8s %6s|%4s %4s|%5s %3s|%3s %3s %3s %3s %3s|%s",
		lib.FormatSeconds(r.runtime),
		lib.FormatPct(lib.MyDivide(r.runtime, totals.runtime)),
		lib.FormatSeconds(r.sleeptime),
		lib.FormatPct(lib.MyDivide(r.sleeptime, totals.sleeptime)),
		lib.FormatCounter(int(r.connections), 4),
		lib.FormatCounter(int(r.active), 4),
		lib.FormatCounter(int(r.hosts), 5),
		lib.FormatCounter(int(r.dbs), 3),
		lib.FormatCounter(int(r.selects), 3),
		lib.FormatCounter(int(r.inserts), 3),
		lib.FormatCounter(int(r.updates), 3),
		lib.FormatCounter(int(r.deletes), 3),
		lib.FormatCounter(int(r.other), 3),
		r.username)
}

// generate a row of totals from a table
func (t PlByUserRows) totals() PlByUserRow {
	var totals PlByUserRow
	totals.username = "Totals"

	for i := range t {
		totals.runtime += t[i].runtime
		totals.sleeptime += t[i].sleeptime
		totals.connections += t[i].connections
		totals.active += t[i].active
		//	totals.hosts += t[i].hosts	This needs to be done differently to get the total number of distinct hosts
		//	totals.dbs += t[i].dbs		This needs to be done differently to get the total number of distinct dbs
		totals.selects += t[i].selects
		totals.inserts += t[i].inserts
		totals.updates += t[i].updates
		totals.deletes += t[i].deletes
		totals.other += t[i].other
	}

	return totals
}

// Headings provides a heading for the rows
func (t PlByUserRows) Headings() string {
	var r PlByUserRow
	return r.headings()
}

// describe a whole row
func (r PlByUserRow) String() string {
	return fmt.Sprintf("%v %v %v %v %v %v %v %v %v %v %v %v", r.runtime, r.connections, r.sleeptime, r.active, r.hosts, r.dbs, r.selects, r.inserts, r.updates, r.deletes, r.other, r.username)
}

// total time is runtime + sleeptime
func (r PlByUserRow) totalTime() uint64 {
	return r.runtime + r.sleeptime
}

// describe a whole table
func (t PlByUserRows) String() string {
	s := ""
	for i := range t {
		s = s + t[i].String() + "\n"
	}
	return s
}

// ByRunTime is for sorting rows by runtime
type ByRunTime PlByUserRows

func (t ByRunTime) Len() int      { return len(t) }
func (t ByRunTime) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t ByRunTime) Less(i, j int) bool {
	return (t[i].totalTime() > t[j].totalTime()) ||
		((t[i].totalTime() == t[j].totalTime()) && (t[i].connections > t[j].connections)) ||
		((t[i].totalTime() == t[j].totalTime()) && (t[i].connections == t[j].connections) && (t[i].username < t[j].username))
}

// Sort by User rows
func (t PlByUserRows) Sort() {
	sort.Sort(ByRunTime(t))
}

func (t PlByUserRows) emptyRowContent() string {
	var r PlByUserRow
	return r.rowContent(r)
}
