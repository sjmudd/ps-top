// Package setupinstruments manages the configuration of
// performance_schema.setupinstruments.
package setupinstruments

import (
	"testing"
)

func TestFuncs(t *testing.T) {

	tests := []struct {
		input    string
		fn       func(string) string
		expected string
	}{
		{"XXXXX", setupInstrumentsFilter, "SELECT NAME, ENABLED, TIMED FROM setup_instruments WHERE NAME LIKE 'XXXXX' AND 'YES' NOT IN (ENABLED,TIMED)"},
		{"YYYYY", collectingSetupInstrumentsMessage, "Collecting setup_instruments YYYYY configuration settings"},
		{"ZZZZZ", updatingSetupInstrumentsMessage, "Updating setup_instruments configuration for: ZZZZZ"},
	}

	for _, test := range tests {
		got := test.fn(test.input)
		if got != test.expected {
			t.Errorf("function fn(%v) returned %v, expected: %v", test.input, got, test.expected)
		}
	}
}

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
		output := expectedError(test.input)
		if output != test.expected {
			t.Errorf("expectedError(%q): got: %v, expected: %v", test.input, output, test.expected)
		}
	}
}
