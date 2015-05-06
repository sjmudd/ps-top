package lib

import (
	"database/sql"
	"log"
)

// Check that select to a table works.  Return an error if we get a failure.
func CheckTableAccess(dbh *sql.DB, table_name string) error {
	sql_select := "SELECT 1 FROM " + table_name + " LIMIT 1"

	var one int
	err := dbh.QueryRow(sql_select).Scan(&one)
	switch {
	case err == sql.ErrNoRows:
		// no rows is unlikely except on a recently started server so take it into account.
		err = nil
	case err != nil:
		log.Fatal("Unable to SELECT FROM "+table_name+":", err)
	default:
		// we don't care if there's no error
	}

	return err
}
