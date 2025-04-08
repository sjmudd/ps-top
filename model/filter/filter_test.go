package filter

import (
	"slices"
	"testing"
)

// TestArgs tests DatabaseFilter.Args
func TestArgs(t *testing.T) {
	tests := []struct {
		given    *DatabaseFilter
		expected []string
	}{
		{nil, []string{}},
		{NewDatabaseFilter("a"), []string{"a"}},
		{NewDatabaseFilter(" a "), []string{"a"}},
		{NewDatabaseFilter("a,b"), []string{"a", "b"}},
		{NewDatabaseFilter(" a, b "), []string{"a", "b"}},
	}

	for _, test := range tests {
		result := test.given.Args()
		if !slices.Equal(test.expected, result) {
			t.Errorf("DatabaseFilter.Args() failed. filter: %+v. Got: %+v, wanted: %+v", test.given, result, test.expected)
		}
	}
}

// TestPlaceholders tests placeholders function
func TestPlaceholders(t *testing.T) {
	tests := []struct {
		given    []string
		expected []string
	}{
		{[]string{}, []string{}},
		{[]string{"A"}, []string{"?"}},
		{[]string{"A", "B"}, []string{"?", "?"}},
	}

	for _, test := range tests {
		result := placeholders(test.given)
		if !slices.Equal(test.expected, result) {
			t.Errorf("placeholders(%v) failed. Got %v, expected %v", test.given, result, test.expected)
		}
	}
}

func TestExtraSQL(t *testing.T) {
	tests := []struct {
		given    *DatabaseFilter
		expected string
	}{
		{NewDatabaseFilter(""), ""},
		{NewDatabaseFilter("a"), "?"},
		{NewDatabaseFilter("a,b"), "?,?"},
		{NewDatabaseFilter("a,b,c"), "?,?,?"},
	}

	for _, test := range tests {
		result := test.given.ExtraSQL()
		expected := ""
		if test.expected != "" {
			expected = ` AND OBJECT_SCHEMA IN (` + test.expected + ")"
		}

		if expected != result {
			t.Errorf("DatabaseFilter.ExtraSQL() failed. filter: %+v. Got: %+v, wanted: %+v", test.given, result, expected)
		}
	}
}
