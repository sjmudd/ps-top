// Package context stores some common information used in various places.
// The name in hindsight might confuse with the golang package context,
// though there is no relation between the two.
package context

import (
	"strings"

	"github.com/sjmudd/ps-top/global"
	"github.com/sjmudd/ps-top/model/filter"
)

// Context holds the common information
type Context struct {
	databaseFilter    *filter.DatabaseFilter
	status            *global.Status
	variables         *global.Variables
	wantRelativeStats bool
}

// NewContext returns the pointer to a new (empty) context
func NewContext(status *global.Status, variables *global.Variables, databaseFilter *filter.DatabaseFilter, wantRelativeStats bool) *Context {
	return &Context{
		databaseFilter:    databaseFilter,
		status:            status,
		variables:         variables,
		wantRelativeStats: wantRelativeStats,
	}
}

// DatabaseFilter returns the database filter to apply on queries (if appropriate)
func (c Context) DatabaseFilter() *filter.DatabaseFilter {
	return c.databaseFilter
}

// Hostname returns the current short hostname
func (c Context) Hostname() string {
	hostname := c.variables.Get("hostname")
	if index := strings.Index(hostname, "."); index >= 0 {
		hostname = hostname[0:index]
	}
	return hostname
}

// MySQLVersion returns the current MySQL version
func (c Context) MySQLVersion() string {
	return c.variables.Get("version")
}

// Uptime returns the time that MySQL has been up (in seconds)
func (c Context) Uptime() int {
	return c.status.Get("Uptime")
}

// Variables returns a pointer to global.Variables
func (c Context) Variables() *global.Variables {
	return c.variables
}

// SetWantRelativeStats tells what we want to see
func (c *Context) SetWantRelativeStats(w bool) {
	c.wantRelativeStats = w
}

// WantRelativeStats tells us what we have asked for
func (c Context) WantRelativeStats() bool {
	return c.wantRelativeStats
}
