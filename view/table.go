// Package table provides a simple way of checking access to a table
package view

import (
	"database/sql"
	"log"

	"github.com/sjmudd/ps-top/mylog"
)

// AccessInfo holds a database and table name and information on whether the table is reachable
type AccessInfo struct {
	database           string
	table              string
	checkedSelectError bool
	selectError        error
}

// NewAccessInfo returns a new AccessInfo type
func NewAccessInfo(database, table string) AccessInfo {
	log.Println("NewAccessInfo(", database, ",", table, ")")
	return AccessInfo{database: database, table: table}
}

// Database returns the database name
func (ta AccessInfo) Database() string {
	return ta.database
}

// Table returns the table name
func (ta AccessInfo) Table() string {
	return ta.table
}

// Name returns the fully qualified table name
func (ta AccessInfo) Name() string {
	if len(ta.database) > 0 && len(ta.table) > 0 {
		return ta.database + "." + ta.table
	}
	return ""
}

// CheckSelectError returns whether SELECT works on the table
func (ta *AccessInfo) CheckSelectError(dbh *sql.DB) error {
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

// SelectError returns the result of ta.selectError
func (ta AccessInfo) SelectError() error {
	if !ta.checkedSelectError {
		mylog.Fatal("table.AccessInfo.SelectError(", ta, ") called without having called CheckSelectError() first")
	}
	return ta.selectError
}
