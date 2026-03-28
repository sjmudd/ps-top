package model

import "database/sql"

// QueryExecutor abstracts database query operations.
// It allows models to work with any database-like entity, not just *sql.DB.
// This improves testability because tests can provide a mock implementation.
type QueryExecutor interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
}
