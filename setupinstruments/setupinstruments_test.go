// Package setupinstruments manages the configuration of
// performance_schema.setupinstruments.
package setupinstruments

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
	for _, test := range tests {
		output := isExpectedError(test.input)
		if output != test.expected {
			t.Errorf("isExpectedError(%q): expected: %v, got: %v", test.input, test.expected, output)
		}
	}
}
