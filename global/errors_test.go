package global

import (
	"errors"
	"testing"
)

func TestError(t *testing.T) {
	tests := []struct {
		errnum   int
		errstr   string
		expected bool
	}{
		{0, "", false},
		{0, "whatever", false},
		{0, "Error 0000 (?????):", true},
		{3167, "Error 3167 (?????): The 'INFORMATION_SCHEMA.GLOBAL_VARIABLES' feature is disabled; see the documentation for 'show_compatibility_56", true},
		{1109, "Error 1109 (42S02): Unknown table 'GLOBAL_VARIABLES' in information_schema", true},
	}
	for _, test := range tests {
		err := errors.New(test.errstr)
		got := IsMysqlError(err, test.errnum)
		if got != test.expected {
			t.Errorf("IsMysqlError(%v,%v) failed: expected: %v, got %v",
				test.errstr,
				test.errnum,
				test.expected,
				got)
		}
	}
}
