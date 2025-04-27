// Package userlatency file contains the library routines for managing the
// information_schema.processlist table.
package userlatency

import (
	"database/sql"

	"github.com/sjmudd/anonymiser"
	"github.com/sjmudd/ps-top/log"
)

// ProcesslistRow contains a row from from information_schema.processlist
type ProcesslistRow struct {
	ID      uint64
	user    string
	host    string
	db      string
	command string
	time    uint64
	state   string
	info    string
}

// get the output of I_S.PROCESSLIST - results only used internally
func collect(db *sql.DB) []ProcesslistRow {
	// we collect all information even if it's mainly empty as we may reference it later
	const query = "SELECT ID, USER, HOST, DB, COMMAND, TIME, STATE, INFO FROM INFORMATION_SCHEMA.PROCESSLIST"

	var (
		t        []ProcesslistRow
		id       sql.NullInt64
		user     sql.NullString
		host     sql.NullString
		database sql.NullString
		command  sql.NullString
		time     sql.NullInt64
		state    sql.NullString
		info     sql.NullString
	)

	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		var r ProcesslistRow
		if err := rows.Scan(
			&id,
			&user,
			&host,
			&database,
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
		if database.Valid {
			r.db = database.String
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
	_ = rows.Close()

	return t
}
