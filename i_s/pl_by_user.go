package i_s

import (
	"fmt"
	"sort"

	"github.com/sjmudd/pstop/lib"
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

// a summary row of information taken from information_schema.processlist
type pl_by_user_row struct {
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
type pl_by_user_rows []pl_by_user_row

/*
username      |Run Time   %age|Sleeping      %|Conn Actv|Hosts DBs|Sel Ins Upd Del Oth|
xxxxxxxxxxxxxx|hh:mm:ss 100.0%|hh:mm:ss 100.0%|9999 9999|9999  999|999 999 999 999 999|
*/

func (r *pl_by_user_row) headings() string {
	return fmt.Sprintf("%-14s|%-8s %6s|%-8s %6s|%4s %4s|%5s %3s|%3s %3s %3s %3s %3s|",
		"User", "Run Time", "%", "Sleeping", "%", "Conn", "Actv", "Hosts", "DBs", "Sel", "Ins", "Upd", "Del", "Oth")
}

// generate a printable result
func (r *pl_by_user_row) row_content(totals pl_by_user_row) string {
	var u string
	if len(r.username) == 0 {
		u = ""
	} else if len(r.username) > 14 {
		u = r.username[0:14]
	} else {
		u = r.username
	}
	return fmt.Sprintf("%-14s|%8s %6s|%8s %6s|%4s %4s|%5s %3s|%3s %3s %3s %3s %3s|",
		u,
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
		lib.FormatCounter(int(r.other), 3))
}

// generate a row of totals from a table
func (t pl_by_user_rows) totals() pl_by_user_row {
	var totals pl_by_user_row
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

func (t pl_by_user_rows) Headings() string {
	var r pl_by_user_row
	return r.headings()
}

// describe a whole row
func (r pl_by_user_row) String() string {
	return fmt.Sprintf("%v %v %v %v %v %v %v %v %v %v", r.username, r.runtime, r.connections, r.sleeptime, r.active, r.hosts, r.dbs, r.selects, r.inserts, r.updates, r.deletes, r.other)
}

// total time is runtime + sleeptime
func (r pl_by_user_row) total_time() uint64 {
	return r.runtime + r.sleeptime
}

// describe a whole table
func (t pl_by_user_rows) String() string {
	s := ""
	for i := range t {
		s = s + t[i].String() + "\n"
	}
	return s
}

// for sorting
type ByRunTime pl_by_user_rows

func (t ByRunTime) Len() int      { return len(t) }
func (t ByRunTime) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t ByRunTime) Less(i, j int) bool {
	return (t[i].total_time() > t[j].total_time()) ||
		((t[i].total_time() == t[j].total_time()) && (t[i].connections > t[j].connections)) ||
		((t[i].total_time() == t[j].total_time()) && (t[i].connections == t[j].connections) && (t[i].username < t[j].username))
}

func (t pl_by_user_rows) Sort() {
	sort.Sort(ByRunTime(t))
}

func (t pl_by_user_rows) emptyRowContent() string {
	var r pl_by_user_row
	return r.row_content(r)
}
