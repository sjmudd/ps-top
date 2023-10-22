// Package userlatency file contains the library routines for managing the
// information_schema.processlist table.
package userlatency

import (
	"database/sql"

	"github.com/sjmudd/anonymiser"
	"github.com/sjmudd/ps-top/log"
)

// ProcesslistRows contains a slice of ProcesslistRow
type ProcesslistRows []ProcesslistRow

// get the output of I_S.PROCESSLIST - results only used internally
func collect(dbh *sql.DB) ProcesslistRows {
	// we collect all information even if it's mainly empty as we may reference it later
	const query = "SELECT ID, USER, HOST, DB, COMMAND, TIME, STATE, INFO FROM INFORMATION_SCHEMA.PROCESSLIST"

	var (
		t       ProcesslistRows
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
		var r ProcesslistRow
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
		log.Println("user:", u, ", anonymised:", a)
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
