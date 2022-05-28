package global

import (
	"testing"
)

func TestVariablesGet(t *testing.T) {
	tests := []struct {
		input     string
		variables map[string]string
		expected  string
	}{
		{"", map[string]string{}, ""},                       // always return empty string if searching for empty string
		{"var1", map[string]string{}, ""},                   // return empty string if not found
		{"var1", map[string]string{"var1": "val1"}, "val1"}, // return value if key found
		{"var2", map[string]string{"var1": "val1", "var2": "val2"}, "val2"},
		{"var3", map[string]string{"var1": "val1", "var2": "val2"}, ""},
	}
	for _, test := range tests {
		v := Variables{variables: test.variables}
		got := v.Get(test.input)
		if got != test.expected {
			t.Errorf("Variables.Get(%v) failed: expected: %q, got %q", test.input, test.expected, got)
		}
	}
}

func TestVariablesTable(t *testing.T) {
	tests := []struct {
		input    bool
		expected string
	}{
		{false, informationSchemaGlobalVariables},
		{true, performanceSchemaGlobalVariables},
	}
	for _, test := range tests {
		got := variablesTable(test.input)
		if got != test.expected {
			t.Errorf("variablesTable(%v) returned unexpected value: expected: %q, got %q", test.input, test.expected, got)
		}
	}
}
