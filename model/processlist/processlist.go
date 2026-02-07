// processlist handles processing of the processlist table whether in I_S or P_S
package processlist

import (
	"database/sql"
	"fmt"

	"github.com/sjmudd/anonymiser"
	"github.com/sjmudd/ps-top/log"
)

const selectCountPSProcesslistTableSQL = `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'performance_schema' and table_name = 'processlist'`

// Do we have P_S.processlist table?
func HavePerformanceSchema(db *sql.DB) (bool, error) {
	var count int

	if err := db.QueryRow(selectCountPSProcesslistTableSQL).Scan(&count); err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("COUNT(*) returns no rows: %w", err)
		}
		return false, fmt.Errorf("COUNT(*) returns unexpected error: %w", err)
	}

	log.Printf("HavePerformanceSchema() returns %d", count)
	return count == 1, nil
}

// ProcesslistRow contains a row from from I_S.processlist or P_S.processlist
type ProcesslistRow struct {
	ID      uint64
	User    string
	Host    string
	Db      string
	Command string
	Time    uint64
	State   string
	Info    string
}

// Return the output of P_S or I_S.PROCESSLIST
func Collect(db *sql.DB) []ProcesslistRow {
	// we collect all information even if it's mainly empty as we may reference it later
	const (
		InformationSchemaQuery = "SELECT ID, USER, HOST, DB, COMMAND, TIME, STATE, INFO FROM INFORMATION_SCHEMA.PROCESSLIST"
		PerformanceSchemaQuery = "SELECT ID, USER, HOST, DB, COMMAND, TIME, STATE, INFO FROM performance_schema.processlist"
	)

	// FIXME: consider caching this response as it is not expected to change
	havePS, err := HavePerformanceSchema(db)
	if err != nil {
		log.Fatal(err)
	}

	var query string
	if havePS {
		query = PerformanceSchemaQuery
	} else {
		query = InformationSchemaQuery
	}

	log.Printf("processlist.Collect: query %v", query)

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
		r.User = a
		r.Host = host.String
		if database.Valid {
			r.Db = database.String
		}
		r.Command = command.String
		r.Time = uint64(time.Int64)
		if state.Valid {
			r.State = state.String
		}
		r.Info = info.String
		t = append(t, r)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	_ = rows.Close()

	return t
}
