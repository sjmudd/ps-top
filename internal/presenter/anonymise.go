package presenter

import (
	"github.com/sjmudd/ps-top/internal/anonymise"
	"github.com/sjmudd/ps-top/internal/snapshot"
)

// The kinds of object anonymised, and the prefix of their anonymous
// equivalents, e.g. schema1. A database is a schema.
const (
	schema = "schema"
	table  = "table"
	user   = "user"
)

// systemSchemas and systemUsers are the server's own schemas and
// processlist users, shown as they are when anonymising: they are on every
// server and say nothing private. Accounts such as root are anonymised, as
// whether they are used may be private.
var (
	systemSchemas = []string{"mysql", "performance_schema", "sys", "information_schema"}
	systemUsers   = []string{"system user", "event_scheduler"}
)

// NewAnonymiser returns an Anonymiser for the views, leaving system schemas
// and system users as they are. Use one for a whole run.
func NewAnonymiser() *anonymise.Anonymiser {
	a := anonymise.New()
	for _, s := range systemSchemas {
		a.Exclude(s, schema)
	}
	for _, u := range systemUsers {
		a.Exclude(u, user)
	}
	return a
}

// anonymiseTable anonymises a key of schema and table, e.g. shop.orders to
// schema1.table1. A table in an excluded schema, e.g. mysql.db, and a key
// of another number of parts, e.g. a file label or a path, are returned as
// they are.
func anonymiseTable(a *anonymise.Anonymiser, key string) string {
	parts := snapshot.KeyParts(key)
	if len(parts) != 2 || a.Excluded(parts[0], schema) {
		return snapshot.KeyText(key)
	}
	return a.Anonymise(parts[0], schema) + "." + a.Anonymise(parts[1], table)
}

// anonymiseUser anonymises a key that is a user, e.g. app to user1.
func anonymiseUser(a *anonymise.Anonymiser, key string) string {
	return a.Anonymise(key, user)
}
