package mapper

import "testing"

func TestTrimPrefix(t *testing.T) {
	m := TrimPrefix("stage/sql/")
	for _, tc := range []struct{ key, want string }{
		{"stage/sql/Sending data", "Sending data"},
		{"stage/innodb/alter table", "stage/innodb/alter table"}, // no prefix: as it is
		{"stage/sql/", ""},
	} {
		if got := m(tc.key); got != tc.want {
			t.Errorf("TrimPrefix(%q)(%q) = %q, want %q", "stage/sql/", tc.key, got, tc.want)
		}
	}
}
