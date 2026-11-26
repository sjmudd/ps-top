package presenter

import (
	"testing"

	"github.com/sjmudd/ps-top/v2/internal/snapshot"
)

// TestNewAnonymiser checks system schemas, their tables and system users are
// shown as they are, and take no numbers from other names.
func TestNewAnonymiser(t *testing.T) {
	a := NewAnonymiser()
	k := snapshot.JoinKey
	tables := []struct{ key, want string }{
		{k("mysql", "db"), "mysql.db"},
		{k("sys", "sys_config"), "sys.sys_config"},
		{k("performance_schema", "threads"), "performance_schema.threads"},
		{k("information_schema", "tables"), "information_schema.tables"},
		{k("shop", "orders"), "schema1.table1"},
		{k("shop", "db"), "schema1.table2"}, // a table named as a system table is anonymised
		{k("mysqlx", "t"), "schema2.table3"},
		{"<redo_log>", "<redo_log>"},
	}
	for _, tt := range tables {
		if got := anonymiseTable(a, tt.key); got != tt.want {
			t.Errorf("anonymiseTable(%q) = %q, want %q", snapshot.KeyText(tt.key), got, tt.want)
		}
	}
	users := []struct{ name, want string }{
		{"system user", "system user"},
		{"event_scheduler", "event_scheduler"},
		{"root", "user1"},
		{"app", "user2"},
	}
	for _, tt := range users {
		if got := anonymiseUser(a, tt.name); got != tt.want {
			t.Errorf("anonymiseUser(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
