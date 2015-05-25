// Package lib provides various library routines
package lib

import (
	"database/sql"
	"log"
)

// CheckTableAccess checks that we have SELECT grants on the table.  Return an error if we get a failure.
func CheckTableAccess(dbh *sql.DB, table string) error {
	sqlSelect := "SELECT 1 FROM " + table + " LIMIT 1"

	var one int
	err := dbh.QueryRow(sqlSelect).Scan(&one)
	switch {
	case err == sql.ErrNoRows:
		// no rows is unlikely except on a recently started server so take it into account.
		err = nil
	case err != nil:
		log.Fatal("Unable to SELECT FROM "+table+":", err)
	default:
		// we don't care if there's no error
	}

	return err
}
