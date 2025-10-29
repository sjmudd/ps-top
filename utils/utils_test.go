package utils

import (
	"math"
	"slices"
	"testing"
)

func TestProgName(t *testing.T) {
	const expected = "utils.test"
	if ProgName != expected {
		t.Errorf("ProgName expected to be %v but actually was %v", expected, ProgName)
	}
}

func TestMyround(t *testing.T) {
	tests := []struct {
		input    float64
		width    int
		decimals int
		expected string
	}{
		{0, 10, 0, "         0"},
		{99.9, 10, 0, "       100"},
		{99.99, 10, 0, "       100"},
		{99.99, 10, 2, "     99.99"},
		{99.999, 10, 0, "       100"},
		{100, 10, 0, "       100"},
		{100.01, 10, 0, "       100"},
		{100.1, 10, 0, "       100"},
		{123, 8, 3, " 123.000"},
		{123, 9, 3, "  123.000"},
		{123, 10, 3, "   123.000"},
	}
	for _, test := range tests {
		got := myround(test.input, test.width, test.decimals)
		if got != test.expected {
			t.Errorf("myformat(%v,%v,%v) failed: expected: %q, got %q", test.input, test.width, test.decimals, test.expected, got)
		}
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		picoseconds uint64
		expected    string
	}{
		// Zero case
		{0, ""},

		// Basic units
		{1, "1 ps"},
		{999, "999 ps"},

		// Nanosecond range
		{1000, "   1.00 ns"},
		{999000, " 999.00 ns"},

		// Microsecond range
		{1000000, "   1.00 us"},
		{999000000, " 999.00 us"},

		// Millisecond range
		{1000000000, "   1.00 ms"},
		{999000000000, " 999.00 ms"},

		// Second range
		{1000000000000, "    1.00 s"},
		{59000000000000, "   59.00 s"},

		// Minute range
		{60000000000000, "    1.00 m"},
		{3540000000000000, "   59.00 m"},

		// Hour range
		{3600000000000000, "    1.00 h"},
		{7200000000000000, "    2.00 h"},
		{3600000000000000 * 24, "   24.00 h"}, // 1 day
	}
	for _, test := range tests {
		got := FormatTime(test.picoseconds)
		if got != test.expected {
			t.Errorf("FormatTime(%v) failed: expected: %q, got %q", test.picoseconds, test.expected, got)
		}
	}
}

func TestSecToTime(t *testing.T) {
	tests := []struct {
		seconds  uint64
		expected string
	}{
		// Zero case
		{0, "00:00:00"},

		// Basic cases
		{1, "00:00:01"},
		{59, "00:00:59"},
		{60, "00:01:00"},
		{61, "00:01:01"},
		{3599, "00:59:59"},
		{3600, "01:00:00"},
		{3601, "01:00:01"},

		// Hour rollovers
		{7199, "01:59:59"},
		{7200, "02:00:00"},
		{86399, "23:59:59"},  // One day minus 1 second
		{86400, "24:00:00"},  // One day
		{90000, "25:00:00"},  // Beyond 24 hours
		{172800, "48:00:00"}, // 2 days

		// Large numbers
		{356400, "99:00:00"},   // 99 hours
		{1000000, "277:46:40"}, // Large number test
	}
	for _, test := range tests {
		got := secToTime(test.seconds)
		if got != test.expected {
			t.Errorf("secToTime(%v) failed: expected: %q, got %q", test.seconds, test.expected, got)
		}
	}
}

func TestDivide(t *testing.T) {
	tests := []struct {
		a        uint64
		b        uint64
		expected float64
	}{
		{1, 0, 0},
		{1, 1, 1},
		{1, 2, 0.5},
		{2, 0, 0},
		{2, 1, 2},
		{2, 2, 1},
		{2, 3, 0.6666666666666666},
	}
	for _, test := range tests {
		got := Divide(test.a, test.b)
		if got != test.expected {
			t.Errorf("Divide(%v,%v) failed: expected: %v, got %v", test.a, test.b, test.expected, got)
		}
	}
}

func TestQualifiedTableName(t *testing.T) {
	tests := []struct {
		schema   string
		table    string
		expected string
	}{
		{"", "", ""},
		{"", "table", "table1"},
		{"schema", "", "schema1"},
		{"schema", "table", "schema1.table1"},
		{"some_schema", "table", "schema2.table1"},
		{"some_schema", "some_table", "schema2.table2"},
		{"test_schema", "test_table", "schema3.table3"},
		{"", "test_table", "table3"},
		{"test_schema", "", "schema3"},
	}

	for _, test := range tests {
		got := QualifiedTableName(test.schema, test.table)
		if got != test.expected {
			t.Errorf("QualifiedTable(%q,%q) failed: expected: %q, got %q", test.schema, test.table, test.expected, got)
		}
	}
}

func TestFormatPct(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		// Zero and near-zero cases
		{0, "      "},
		{0.000001, "      "},
		{0.00009, "      "},

		// Normal cases with rounding
		{0.0049, "  0.5%"},
		{0.005, "  0.5%"},
		{0.0051, "  0.5%"},
		{0.049, "  4.9%"},
		{0.05, "  5.0%"},
		{0.051, "  5.1%"},
		{0.49, " 49.0%"},
		{0.5, " 50.0%"},
		{0.51, " 51.0%"},
		{1, "100.0%"},

		// Edge cases and overflow
		{9.5, "950.0%"},
		{9.999, "999.9%"},  // Last value before overflow
		{10.001, "+++.+%"}, // First overflow value
		{100, "+++.+%"},    // Large overflow
		{1000, "+++.+%"},   // Very large overflow
	}

	for _, test := range tests {
		got := FormatPct(test.input)
		if got != test.expected {
			t.Errorf("FormatPct(%v) failed: expected: %q, got %q", test.input, test.expected, got)
		}
	}
}

func TestFormatAmount(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, ""},
		{1, "1"},
		{1024 - 1, "1023"},
		{1024, "1024"},
		{1024 + 1, "  1.00 k"},
		{1024 * 1024, "1024.0 k"},
		{1024 * 1024 * 1024, "1024.0 M"},
		{1024 * 1024 * 1024 * 1024, "1024.0 G"},
	}

	for _, test := range tests {
		got := FormatAmount(test.input)
		if got != test.expected {
			t.Errorf("FormatAmount(%v) failed: expected: %q, got %q", test.input, test.expected, got)
		}
	}
}

func TestFormatCounter(t *testing.T) {
	tests := []struct {
		counter  int
		width    int
		expected string
	}{
		// Zero cases with different widths
		{0, 1, " "},
		{0, 5, "     "},
		{0, 6, "      "},

		// Basic positive numbers
		{1, 5, "    1"},
		{1, 6, "     1"},

		// Numbers at width boundaries
		{999, 3, "999"},
		{1000, 3, "1000"}, // Exceeds width
		{1000, 6, "  1000"},
		{9999, 4, "9999"},
		{10000, 4, "10000"}, // Exceeds width
		{10000, 6, " 10000"},
		{100000, 6, "100000"},
		{1000000, 6, "1000000"},
		{1000000, 7, "1000000"},

		// Negative numbers
		{-1, 5, "   -1"},
		{-10, 5, "  -10"},
		{-100, 5, " -100"},
		{-1000, 6, " -1000"},
		{-10000, 6, "-10000"},

		// Width smaller than number
		{1234, 2, "1234"},
		{-1234, 2, "-1234"},

		// Very large numbers
		{math.MaxInt32, 10, "2147483647"},
		{math.MinInt32, 11, "-2147483648"},
	}

	for _, test := range tests {
		got := FormatCounter(test.counter, test.width)
		if got != test.expected {
			t.Errorf("FormatCounter(%v,%v) failed: expected: %q, got %q", test.counter, test.width, test.expected, got)
		}
	}
}

func TestDuplicateSlice(t *testing.T) {
	// Empty slice
	test1 := []int{}
	got1 := DuplicateSlice(test1)
	if !slices.Equal(test1, got1) {
		t.Errorf("DuplicateSlice(%v) failed. Got: %+v", test1, got1)
	}

	// Basic types
	test2 := []int{1, 2, 3}
	got2 := DuplicateSlice(test2)
	if !slices.Equal(test2, got2) {
		t.Errorf("DuplicateSlice(%v) failed. Got: %+v", test2, got2)
	}

	test3 := []string{"a", "b", "c"}
	got3 := DuplicateSlice(test3)
	if !slices.Equal(test3, got3) {
		t.Errorf("DuplicateSlice(%v) failed. Got: %+v", test3, got3)
	}

	// Mixed types
	test4 := []any{"a", "b", "c", 1, 2, 3}
	got4 := DuplicateSlice(test4)
	if !slices.Equal(test4, got4) {
		t.Errorf("DuplicateSlice(%v) failed. Got: %+v", test4, got4)
	}

	// Slice with nil elements
	test5 := []any{"a", nil, "c", nil, 3}
	got5 := DuplicateSlice(test5)
	if !slices.Equal(test5, got5) {
		t.Errorf("DuplicateSlice(%v) failed. Got: %+v", test5, got5)
	}

	// Nested slices
	test6 := [][]int{{1, 2}, {3, 4}, {5, 6}}
	got6 := DuplicateSlice(test6)
	for i := range test6 {
		if !slices.Equal(test6[i], got6[i]) {
			t.Errorf("DuplicateSlice nested slice at index %d failed. Expected %v, got %v", i, test6[i], got6[i])
		}
	}

	// Large slice
	largeSlice := make([]int, 10000)
	for i := range largeSlice {
		largeSlice[i] = i
	}
	got7 := DuplicateSlice(largeSlice)
	if !slices.Equal(largeSlice, got7) {
		t.Errorf("DuplicateSlice(large slice) failed")
	}
}

func TestSignedFormatAmount(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, ""},
		{1, "1"},
		{-1, "-1"},
		{1023, "1023"},
		{-1023, "-1023"},
		{1024, "1024"},
		{-1024, "-1024"},
		{1025, "  1.00 k"},
		{-1025, " -1.00 k"},
		{1024 * 1024, "1024.0 k"},
		{-1024 * 1024, "-1024.0 k"},
		{1024 * 1024 * 1024, "1024.0 M"},
		{-1024 * 1024 * 1024, "-1024.0 M"},
		{1024 * 1024 * 1024 * 1024, "1024.0 G"},
		{-1024 * 1024 * 1024 * 1024, "-1024.0 G"},
		// Edge cases
		{9223372036854775807, "8388608.0 P"},   // max int64
		{-9223372036854775808, "-8388608.0 P"}, // min int64
	}

	for _, test := range tests {
		got := SignedFormatAmount(test.input)
		if got != test.expected {
			t.Errorf("SignedFormatAmount(%v) failed: expected: %q, got %q", test.input, test.expected, got)
		}
	}
}

func TestSignedDivide(t *testing.T) {
	tests := []struct {
		a        int64
		b        int64
		expected float64
	}{
		{0, 1, 0},
		{1, 0, 0},
		{-1, 0, 0},
		{1, 1, 1},
		{-1, 1, -1},
		{1, -1, -1},
		{-1, -1, 1},
		{100, 2, 50},
		{-100, 2, -50},
		{100, -2, -50},
		{-100, -2, 50},
		{9223372036854775807, 1, 9223372036854775807},   // max int64
		{-9223372036854775808, 1, -9223372036854775808}, // min int64
		{9223372036854775807, -1, -9223372036854775807},
		{-9223372036854775808, -1, 9223372036854775808},
	}

	for _, test := range tests {
		got := SignedDivide(test.a, test.b)
		if got != test.expected {
			t.Errorf("SignedDivide(%v,%v) failed: expected: %v, got %v", test.a, test.b, test.expected, got)
		}
	}
}
