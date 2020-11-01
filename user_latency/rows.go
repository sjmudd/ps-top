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

// Rows contains a slice of Row
type Rows []Row

// get the output of I_S.PROCESSLIST
func collect(dbh *sql.DB) Rows {
	// we collect all information even if it's mainly empty as we may reference it later
	const query = "SELECT ID, USER, HOST, DB, COMMAND, TIME, STATE, INFO FROM INFORMATION_SCHEMA.PROCESSLIST"

	var (
		t       Rows
		id      sql.NullInt64
		user    sql.NullString
		host    sql.NullString
		db      sql.NullString
		command sql.NullString
		time    sql.NullInt64
		state   sql.NullString
		info    sql.NullString
	)

	rows, err := dbh.Query(query)
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

// describe a whole table
func (t Rows) String() string {
	return fmt.Sprintf("FIXME output of i_s")
}
