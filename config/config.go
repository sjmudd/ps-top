// Package context stores some common information used in various places.
// The name in hindsight might confuse with the golang package context,
// though there is no relation between the two.
package config

import (
	"strings"

	"github.com/sjmudd/anonymiser"
	"github.com/sjmudd/ps-top/global"
	"github.com/sjmudd/ps-top/model/filter"
)

// Config holds the common information
type Config struct {
	databaseFilter    *filter.DatabaseFilter
	status            *global.Status
	variables         *global.Variables
	wantRelativeStats bool
}

// NewConfig returns the pointer to a new (empty) context
func NewConfig(status *global.Status, variables *global.Variables, databaseFilter *filter.DatabaseFilter, wantRelativeStats bool) *Config {
	return &Config{
		databaseFilter:    databaseFilter,
		status:            status,
		variables:         variables,
		wantRelativeStats: wantRelativeStats,
	}
}

// DatabaseFilter returns the database filter to apply on queries (if appropriate)
func (c Config) DatabaseFilter() *filter.DatabaseFilter {
	return c.databaseFilter
}

// Hostname returns the current short hostname
func (c Config) Hostname() string {
	hostname := anonymiser.Anonymise("hostname", c.variables.Get("hostname"))
	if index := strings.Index(hostname, "."); index >= 0 {
		hostname = hostname[0:index]
	}
	return hostname
}

// MySQLVersion returns the current MySQL version
func (c Config) MySQLVersion() string {
	return c.variables.Get("version")
}

// Uptime returns the time that MySQL has been up (in seconds)
func (c Config) Uptime() int {
	return c.status.Get("Uptime")
}

// Variables returns a pointer to global.Variables
func (c Config) Variables() *global.Variables {
	return c.variables
}

// SetWantRelativeStats tells what we want to see
func (c *Config) SetWantRelativeStats(w bool) {
	c.wantRelativeStats = w
}

// WantRelativeStats tells us what we have asked for
func (c Config) WantRelativeStats() bool {
	return c.wantRelativeStats
}
