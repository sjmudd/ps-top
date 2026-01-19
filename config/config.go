// Package config stores some common information used in various places.
// The name in hindsight might confuse with the golang package config,
// though there is no relation between the two.
package config

import (
	"strings"
	"time"

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
	uptime            int
	uptimeEpoch       int64
}

// NewConfig returns the pointer to a new (empty) config
func NewConfig(status *global.Status, variables *global.Variables, databaseFilter *filter.DatabaseFilter, wantRelativeStats bool) *Config {
	return &Config{
		databaseFilter:    databaseFilter,
		status:            status,
		variables:         variables,
		wantRelativeStats: wantRelativeStats,
		uptime:            0,
		uptimeEpoch:       0,
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
func (c *Config) Uptime() int {

	// A previous version of this function was systematically returning c.status.Get("Uptime").
	// This generated three (3) queries to MySQL on each call (including Prepare and Close stmt).
	// We now only query MySQL on the 1st call, computing uptime on the next calls.

	if c.uptime == 0 {
		c.uptime = c.status.Get("Uptime")
		c.uptimeEpoch = time.Now().Unix()
		return c.uptime
	} else {
		return c.uptime + (int)(time.Now().Unix() - c.uptimeEpoch)
	}
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
