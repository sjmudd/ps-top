// Package view provides a simple way of checking access to a table
package view

import (
	"database/sql"

	"github.com/sjmudd/ps-top/log"
)

// AccessInfo holds a database and table name and information on whether the table is reachable
type AccessInfo struct {
	Database           string
	Table              string
	checkedSelectError bool
	selectError        error
}

// NewAccessInfo returns a new AccessInfo type
func NewAccessInfo(database, table string) AccessInfo {
	log.Println("NewAccessInfo(", database, ",", table, ")")
	return AccessInfo{Database: database, Table: table}
}

// Name returns the fully qualified table name
func (ta AccessInfo) Name() string {
	if ta.Database == "performance_schema" && len(ta.Table) > 0 {
		// no need to prefix the table with the ps schema, this is the default schema.
		return ta.Table
	}
	if len(ta.Database) > 0 && len(ta.Table) > 0 {
		return ta.Database + "." + ta.Table
	}
	return ""
}

// CheckSelectError returns whether SELECT works on the table
func (ta *AccessInfo) CheckSelectError(db *sql.DB) error {
	// return cached result if we have one
	if ta.checkedSelectError {
		return ta.selectError
	}

	var one int
	err := db.QueryRow("SELECT 1 FROM " + ta.Name() + " LIMIT 1").Scan(&one)

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
		log.Fatal("table.AccessInfo.SelectError(", ta, ") called without having called CheckSelectError() first")
	}
	return ta.selectError
}
