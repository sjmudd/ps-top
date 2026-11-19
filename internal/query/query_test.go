package query

import "testing"

func TestRunInvalidType(t *testing.T) {
	// The type check runs before the database is used, so a nil db is safe.
	if err := Run(nil, "SELECT 1", []Column{{"a", Type(9)}}, func(Row) {}); err == nil {
		t.Error("invalid type: no error")
	}
}
