// Package context stores some common information used in various places
package context

import (
	"strings"
	"time"

	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/version"
)

// Context holds the common information
type Context struct {
	hostname          string
	mysqlVersion      string
	wantRelativeStats bool
	version           string
	last              time.Time
	uptime            int
}

// NewContext returns the pointer to a new (empty) context
func NewContext() *Context {
	return new(Context)
}

// SetHostname sets the server's current hostname
// - we actually want the shortname - for now store that way but a "view" should show the short version?
func (c *Context) SetHostname(hostname string) {
	if index := strings.Index(hostname, "."); index >= 0 {
		hostname = hostname[0:index]
	}
	c.hostname = hostname
}

// SetMySQLVersion stores this value to be used later (we assume it doesn't change)
func (c *Context) SetMySQLVersion(version string) {
	c.mysqlVersion = version
}

// SetWantRelativeStats is also used frequently and we set the current intention here
func (c *Context) SetWantRelativeStats(wantRelativeStats bool) {
	c.wantRelativeStats = wantRelativeStats
}

// Hostname returns the current hostname
func (c Context) Hostname() string {
	return c.hostname
}

// MySQLVersion returns the current MySQL version
func (c Context) MySQLVersion() string {
	return c.mysqlVersion
}

// WantRelativeStats indicates if we want this or not
func (c Context) WantRelativeStats() bool {
	return c.wantRelativeStats
}

// Version returns the Application version
func (c Context) Version() string {
	return version.Version()
}

// MyName returns the program's name
func (c Context) MyName() string {
	return lib.MyName()
}

// SetUptime records the time mysql has been up
func (c *Context) SetUptime(uptime int) {
	c.uptime = uptime
}

// Uptime returns the time that MySQL has been up
func (c Context) Uptime() int {
	return c.uptime
}
