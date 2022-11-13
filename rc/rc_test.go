package rc

import (
	"testing"
)

// Munge Optionally munges table names so they can be combined.
// - this reads ~/.pstoprc for configuration information.
// - e.g.
// [munge]
// <re_match> = <replace>
// _[0-9]{8}$ = _YYYYMMDD
// _[0-9]{6}$ = _YYYYMM
func TestMunge(t *testing.T) {
	type Config = struct {
		regex       string
		replacement string
	}

	tests := []struct {
		input    string
		expected string
		config   []Config
	}{
		{"", "", nil},             // empty input, config
		{"999999", "999999", nil}, // empty config
		{"999999", "111111", []Config{{"999999", "111111"}}},                                                 // crude replacement
		{"999999", "111111", []Config{{"99..99", "111111"}}},                                                 // less crude replacement
		{"999999", "999999", []Config{{"_[0-9]{8}$", "_YYYYMMDD"}, {"_[0-9]{6}$", "_YYYYMM"}}},               // non-matching pattern against 2 config entries
		{"test_20221113", "test_YYYYMMDD", []Config{{"_[0-9]{8}$", "_YYYYMMDD"}, {"_[0-9]{6}$", "_YYYYMM"}}}, // Year/month/day match with 2 config entries
		{"test_202211", "test_YYYYMM", []Config{{"_[0-9]{8}$", "_YYYYMMDD"}, {"_[0-9]{6}$", "_YYYYMM"}}},     // Year/month match with 2 config entries
	}

	for _, test := range tests {
		// load the test data
		regexps = make([]mungeRegexp, 0, len(test.config))
		for _, config := range test.config {
			addPattern(config.regex, config.replacement)
		}
		loaded = true

		result := Munge(test.input)
		if test.expected != result {
			t.Errorf("Munge(%v) failed: got %q, expected: %q", test.input, result, test.expected)
		}
	}
}
