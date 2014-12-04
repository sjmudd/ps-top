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
		log.Println("No setting with that variable_name", one)
	case err != nil:
		log.Fatal(err)
	default:
		// we don't care if there's no error
	}

	return err
}
