// Package user_latency file contains the library routines for managing the
// information_schema.processlist table.
package user_latency

import (
	"fmt"
)

/*

CREATE TEMPORARY TABLE `PROCESSLIST` (
	`ID` bigint(21) unsigned NOT NULL DEFAULT '0',
	`USER` varchar(16) NOT NULL DEFAULT '',
	`HOST` varchar(64) NOT NULL DEFAULT '',
	`DB` varchar(64) DEFAULT NULL, `COMMAND` varchar(16) NOT NULL DEFAULT '', `TIME` int(7) NOT NULL DEFAULT '0', `STATE` varchar(64) DEFAULT NULL,
	`INFO` longtext
) ENGINE=MyISAM DEFAULT CHARSET=utf8

*/

// Row contains a row from from information_schema.processlist
type Row struct {
	ID      uint64
	user    string
	host    string
	db      string
	command string
	time    uint64
	state   string
	info    string
}

// describe a whole row
func (r Row) String() string {
	return fmt.Sprintf("FIXME otuput of i_s")
}
