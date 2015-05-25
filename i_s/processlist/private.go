// Package processlist file contains the library routines for managing the
// information_schema.processlist table.
package processlist

import (
	"database/sql"
	"fmt"
	"log"
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

// a row from information_schema.processlist
type tableRow struct {
	ID      uint64
	USER    string
	HOST    string
	DB      string
	COMMAND string
	TIME    uint64
	STATE   string
	INFO    string
}
type tableRows []tableRow

// get the output of I_S.PROCESSLIST
func selectRows(dbh *sql.DB) tableRows {
	var t tableRows
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
		var r tableRow
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
		r.USER = user.String
		r.HOST = host.String
		if db.Valid {
			r.DB = db.String
		}
		r.COMMAND = command.String
		r.TIME = uint64(time.Int64)
		if state.Valid {
			r.STATE = state.String
		}
		r.INFO = info.String
		t = append(t, r)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return t
}

// describe a whole row
func (r tableRow) String() string {
	return fmt.Sprintf("FIXME otuput of i_s")
}

// describe a whole table
func (t tableRows) String() string {
	return fmt.Sprintf("FIXME otuput of i_s")
}
