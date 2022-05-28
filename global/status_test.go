package global

import (
	"testing"
)

func TestStatusTable(t *testing.T) {
	tests := []struct {
		input    bool
		expected string
	}{
		{false, informationSchemaGlobalStatus},
		{true, performanceSchemaGlobalStatus},
	}
	for _, test := range tests {
		got := statusTable(test.input)
		if got != test.expected {
			t.Errorf("statusTable(%v) returned unexpected value: expected: %q, got %q", test.input, test.expected, got)
		}
	}
}
