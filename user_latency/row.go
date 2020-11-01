// Package user_latency file contains the library routines for managing the
// information_schema.processlist table.
package user_latency

// Row contains a row from from information_schema.processlist
type Row struct {
	ID      uint64
	user    string
	host    string
	db      string
	command string
	time    uint64
	state   string
	info    string
}
