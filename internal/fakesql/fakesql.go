// Package fakesql is a database/sql driver for tests: every query returns the
// same fixed result, or fails with a fixed error. It lets code that takes a
// *sql.DB be tested without a server.
package fakesql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sync/atomic"
)

// Result is what every query returns: Err if set, else Columns and Rows.
// A row has one value per column: int64, string or nil (NULL).
type Result struct {
	Columns []string
	Rows    [][]driver.Value
	Err     error
}

// DB is a *sql.DB whose queries return a fixed Result.
type DB struct {
	*sql.DB
	queries atomic.Int64
}

// Open returns a DB answering every query with r.
func Open(r Result) *DB {
	d := &DB{}
	d.DB = sql.OpenDB(connector{result: r, queries: &d.queries})
	return d
}

// Queries returns how many queries were run.
func (d *DB) Queries() int64 { return d.queries.Load() }

type connector struct {
	result  Result
	queries *atomic.Int64
}

func (c connector) Connect(context.Context) (driver.Conn, error) { return conn(c), nil }
func (c connector) Driver() driver.Driver                        { return fakeDriver{} }

type fakeDriver struct{}

func (fakeDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("fakesql: use Open")
}

type conn connector

func (c conn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	c.queries.Add(1)
	if c.result.Err != nil {
		return nil, c.result.Err
	}
	return &rows{result: c.result}, nil
}

func (conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("fakesql: only queries without prepare are supported")
}
func (conn) Close() error { return nil }
func (conn) Begin() (driver.Tx, error) {
	return nil, errors.New("fakesql: transactions are not supported")
}

type rows struct {
	result Result
	next   int
}

func (r *rows) Columns() []string { return r.result.Columns }
func (r *rows) Close() error      { return nil }

func (r *rows) Next(dest []driver.Value) error {
	if r.next >= len(r.result.Rows) {
		return io.EOF
	}
	copy(dest, r.result.Rows[r.next])
	r.next++
	return nil
}
