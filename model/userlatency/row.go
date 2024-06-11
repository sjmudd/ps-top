// Package userlatency manages the output from INFORMATION_SCHEMA.PROCESSLIST
package userlatency

/*

CREATE TEMPORARY TABLE `PROCESSLIST` (
  `ID` bigint unsigned NOT NULL DEFAULT '0',
  `USER` varchar(32) NOT NULL DEFAULT '',
  `HOST` varchar(261) NOT NULL DEFAULT '',
  `DB` varchar(64) DEFAULT NULL,
  `COMMAND` varchar(16) NOT NULL DEFAULT '',
  `TIME` int NOT NULL DEFAULT '0',
  `STATE` varchar(64) DEFAULT NULL,
  `INFO` longtext
) ENGINE=InnoDB DEFAULT CHARSET=utf8

*/

// Row contains a summary row of information taken from information_schema.processlist
type Row struct {
	Username    string
	Runtime     uint64
	Sleeptime   uint64
	Connections uint64
	Active      uint64
	Hosts       uint64
	Dbs         uint64
	Selects     uint64
	Inserts     uint64
	Updates     uint64
	Deletes     uint64
	Other       uint64
}

// TotalTime returns Runtime + Sleeptime
func (r Row) TotalTime() uint64 {
	return r.Runtime + r.Sleeptime
}

// totals returns the totals of all rows
func totals(rows []Row) Row {
	total := Row{Username: "Totals"}

	for _, row := range rows {
		total.Runtime += row.Runtime
		total.Sleeptime += row.Sleeptime
		total.Connections += row.Connections
		total.Active += row.Active
		total.Selects += row.Selects
		total.Inserts += row.Inserts
		total.Updates += row.Updates
		total.Deletes += row.Deletes
		total.Other += row.Other
	}

	return total
}
