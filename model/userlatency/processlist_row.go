// Package userlatency file contains the library routines for managing the
// information_schema.processlist table.
package userlatency

// ProcesslistRow contains a row from from information_schema.processlist
type ProcesslistRow struct {
	ID      uint64
	user    string
	host    string
	db      string
	command string
	time    uint64
	state   string
	info    string
}
