// Package user_latency file contains the library routines for managing the
// information_schema.processlist table.
package user_latency

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/sjmudd/anonymiser"
	"github.com/sjmudd/ps-top/logger"
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

// Rows contains a slice of Row
type Rows []Row

// get the output of I_S.PROCESSLIST
func selectRows(dbh *sql.DB) Rows {
	var t Rows
	var id sql.NullInt64
	var user sql.NullString
	var host sql.NullString
	var db sql.NullString
	var command sql.NullString
	var time sql.NullInt64
	var state sql.NullString
	var info sql.NullString

	// we collect all information even if it's mainly empty as we may reference it later

	sql := "SELECT ID, USER, HOST, DB, COMMAND, TIME, STATE, INFO FROM INFORMATION_SCHEMA.PROCESSLIST"

	rows, err := dbh.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var r Row
		if err := rows.Scan(
			&id,
			&user,
			&host,
			&db,
			&command,
			&time,
			&state,
			&info); err != nil {
			log.Fatal(err)
		}
		r.ID = uint64(id.Int64)

		// be verbose for debugging.
		u := user.String
		a := anonymiser.Anonymise("user", user.String)
		logger.Println("user:", u, ", anonymised:", a)
		r.user = a
		r.host = host.String
		if db.Valid {
			r.db = db.String
		}
		r.command = command.String
		r.time = uint64(time.Int64)
		if state.Valid {
			r.state = state.String
		}
		r.info = info.String
		t = append(t, r)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return t
}

// describe a whole row
func (r Row) String() string {
	return fmt.Sprintf("FIXME otuput of i_s")
}

// describe a whole table
func (t Rows) String() string {
	return fmt.Sprintf("FIXME otuput of i_s")
}
