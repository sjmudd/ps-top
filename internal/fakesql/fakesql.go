// Package fakesql is a database/sql driver for tests. Each statement is
// answered by a handler, or with one fixed result, and every statement run is
// recorded. It lets code that takes a *sql.DB be tested without a server.
package fakesql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sync"
)

// Statement is one statement run: its text, its arguments, and whether it
// was run with Exec (else with Query).
type Statement struct {
	Query string
	Args  []driver.Value
	Exec  bool
}

// Result is the answer to a statement: Err if set, else Columns and Rows for
// a query, RowsAffected for Exec. A row has one value per column: int64,
// string or nil (NULL).
type Result struct {
	Columns      []string
	Rows         [][]driver.Value
	RowsAffected int64
	Err          error
}

// Handler answers one statement. It may be called from several goroutines.
type Handler func(Statement) Result

// DB is a *sql.DB whose statements a Handler answers.
type DB struct {
	*sql.DB
	mu   sync.Mutex
	run  []Statement
	hand Handler
}

// Open returns a DB answering every statement with r.
func Open(r Result) *DB {
	return OpenFunc(func(Statement) Result { return r })
}

// OpenFunc returns a DB answering each statement with h.
func OpenFunc(h Handler) *DB {
	d := &DB{hand: h}
	d.DB = sql.OpenDB(connector{d})
	return d
}

// Statements returns the statements run so far, in order.
func (d *DB) Statements() []Statement {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]Statement(nil), d.run...)
}

func (d *DB) answer(query string, args []driver.NamedValue, exec bool) Result {
	s := Statement{Query: query, Exec: exec}
	for _, a := range args {
		s.Args = append(s.Args, a.Value)
	}
	d.mu.Lock()
	d.run = append(d.run, s)
	d.mu.Unlock()
	return d.hand(s)
}

type connector struct{ d *DB }

func (c connector) Connect(context.Context) (driver.Conn, error) { return conn(c), nil }
func (c connector) Driver() driver.Driver                        { return fakeDriver{} }

type fakeDriver struct{}

func (fakeDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("fakesql: use Open or OpenFunc")
}

type conn connector

func (c conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	r := c.d.answer(query, args, false)
	if r.Err != nil {
		return nil, r.Err
	}
	return &rows{result: r}, nil
}

func (c conn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	r := c.d.answer(query, args, true)
	if r.Err != nil {
		return nil, r.Err
	}
	return driver.RowsAffected(r.RowsAffected), nil
}

func (conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("fakesql: only statements without prepare are supported")
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
