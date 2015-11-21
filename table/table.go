// Package table provides a simple way of checking access to a table
package table

import (
	"database/sql"
	"log"

	"github.com/sjmudd/ps-top/logger"
)

// Access holds a database and table name and information on whether the table is reachable
type Access struct {
	database            string
	table               string
	checkedSelectError  bool
	selectError         error
	checkedConfigurable bool
	configurable        bool
}

// NewAccess returns a new Access type
func NewAccess(database, table string) Access {
	logger.Println("NewAccess(", database, ",", table, ")")
	return Access{database: database, table: table}
}

// Database returns the database name
func (ta Access) Database() string {
	return ta.database
}

// Table returns the table name
func (ta Access) Table() string {
	return ta.table
}

// Name returns the fully qualified table name
func (ta Access) Name() string {
	if len(ta.database) > 0 && len(ta.table) > 0 {
		return ta.database + "." + ta.table
	}
	return ""
}

// SelectError returns whether SELECT works on the table
func (ta *Access) CheckSelectError(dbh *sql.DB) error {
	// return cached result if we have one
	if ta.checkedSelectError {
		return ta.selectError
	}

	var one int
	err := dbh.QueryRow("SELECT 1 FROM " + ta.Name() + " LIMIT 1").Scan(&one)

	switch {
	case err == sql.ErrNoRows:
		ta.selectError = nil // no rows is fine
	case err != nil:
		ta.selectError = err // keep this
	default:
		ta.selectError = nil // select worked
	}
	ta.checkedSelectError = true

	return ta.selectError
}

// this hands back whatever it has
func (ta Access) SelectError() error {
	if !ta.checkedSelectError {
		log.Fatal("table.Access.SelectError(", ta, ") called without having called CheckSelectError() first")
	}
	return ta.selectError
}
