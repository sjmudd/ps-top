// Package setup_instruments manages the configuration of
// performance_schema.setup_instruments.
package setup_instruments

import (
	"testing"
)

func TestIsExpectedError(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"Error 0000: some other error message", false},
		{"Error 1142: UPDATE command denied to user 'myuser'@'10.11.12.13' for table 'setup_instruments'", true},
		{"Error 1146: Table 'test.no_such_table' doesn't exist", false},
		{"Error 1290: The MySQL server is running with the --read-only option so it cannot execute this statement", true},
		{"Error 9999: some other error message", false},
	}
	for i := range tests {
		output := isExpectedError(tests[i].input)
		if output != tests[i].expected {
			t.Errorf("isExpectedError(%v): expected: %v, got: %v", tests[i].input, tests[i].expected, output)
		}
	}
}
