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
		{"Error 0000:", false},
		{"Error 1142:", true},
		{"Error 1290:", true},
		{"Error 9999:", false},
	}
	for i := range tests {
		output := isExpectedError(tests[i].input)
		if output != tests[i].expected {
			t.Errorf("isExpectedError(%v): expected:%v, got: %v", tests[i].input, tests[i].expected, output)
		}
	}
}
