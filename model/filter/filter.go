// Package filter contains the routines for managing a database filter.
package filter

import (
	"strings"
)

// DatabaseFilter stores a list of filtered databases given a comma-separated list of database names
type DatabaseFilter struct {
	userInput     string
	filteredInput []string
}

// NewDatabaseFilter returns the DatabaseFilter based on the comma-separated list of database names given
func NewDatabaseFilter(filter string) *DatabaseFilter {
	dbf := &DatabaseFilter{
		userInput:     filter,
		filteredInput: make([]string, 0),
	}

	// whitespace trim the unfiltered and ignore any empty strings or strings with spaces
	for _, name := range strings.Split(filter, ",") {
		name = strings.TrimSpace(name)
		if len(name) > 0 && !strings.Contains(name, " ") {
			dbf.filteredInput = append(dbf.filteredInput, name)
		}
	}
	return dbf
}

// Args returns the arguments to be provided to sql.Query(..., args)
// - if f == nil return an empty slice
func (f *DatabaseFilter) Args() []string {
	if f == nil {
		return []string{}
	}
	return f.filteredInput
}

// return placeholders for each name
func placeholders(names []string) []string {
	var s []string

	for range names {
		s = append(s, "?")
	}
	return s
}

// ExtraSQL returns the extra string to apply to the base SQL statement (placeholders)
func (f *DatabaseFilter) ExtraSQL() string {
	if len(f.filteredInput) == 0 {
		return ""
	}

	return ` AND OBJECT_SCHEMA IN (` + strings.Join(placeholders(f.filteredInput), `,`) + `)`
}
