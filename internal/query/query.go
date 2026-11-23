// Package query runs a SQL statement and hands each result row to the caller,
// with each column scanned by its declared type. It knows nothing about keys,
// kinds or snapshots.
package query

import (
	"database/sql"
	"fmt"
)

// Type says how a result column is scanned.
type Type int

const (
	// String is nullable text.
	String Type = iota
	// Uint is an unsigned bigint.
	Uint
)

// Column describes one result column.
type Column struct {
	Name string
	Type Type
}

// Row is one result row. It is valid only during the call to Run's fn.
type Row struct {
	cols []Column
	strs []sql.NullString
	nums []sql.Null[uint64]
}

// Str returns String column i; NULL gives "".
func (r Row) Str(i int) string { return r.strs[i].String }

// Uint returns Uint column i; NULL gives 0.
func (r Row) Uint(i int) uint64 { return r.nums[i].V }

// Null reports whether column i is NULL.
func (r Row) Null(i int) bool {
	if r.cols[i].Type == String {
		return !r.strs[i].Valid
	}
	return !r.nums[i].Valid
}

// Run executes statement with filteredSchemas, the schema names for its "?"
// placeholders, one each (nil: no filter, and no placeholders), and calls fn
// once per result row, in order. The result must have exactly len(cols)
// columns.
func Run(db *sql.DB, statement string, filteredSchemas []string, cols []Column, fn func(Row)) error {
	for _, c := range cols {
		if c.Type != String && c.Type != Uint {
			return fmt.Errorf("query: column %q has invalid type %d", c.Name, int(c.Type))
		}
	}
	values := make([]any, len(filteredSchemas))
	for i, s := range filteredSchemas {
		values[i] = s
	}
	rs, err := db.Query(statement, values...)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer func() { _ = rs.Close() }()
	names, err := rs.Columns()
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	if len(names) != len(cols) {
		return fmt.Errorf("query: result has %d columns, want %d", len(names), len(cols))
	}
	r := Row{cols: cols, strs: make([]sql.NullString, len(cols)), nums: make([]sql.Null[uint64], len(cols))}
	dest := make([]any, len(cols))
	for i, c := range cols {
		if c.Type == String {
			dest[i] = &r.strs[i]
		} else {
			dest[i] = &r.nums[i]
		}
	}
	for rs.Next() {
		if err := rs.Scan(dest...); err != nil {
			return fmt.Errorf("query: %w", err)
		}
		fn(r)
	}
	if err := rs.Err(); err != nil {
		return fmt.Errorf("query: %w", err)
	}
	return nil
}
