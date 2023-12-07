package utils

import (
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
		{0, ""},
		{1, "1 ps"},
		{1000, "   1.00 ns"},
		{1000000, "   1.00 us"},
		{1000000000, "   1.00 ms"},
		{1000000000000, "    1.00 s"},
		{60000000000000, "    1.00 m"},
		{3600000000000000, "    1.00 h"},
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
		{0, "00:00:00"},
		{1, "00:00:01"},
		{60, "00:01:00"},
		{61, "00:01:01"},
		{3600, "01:00:00"},
		{3601, "01:00:01"},
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
		{"schema", "table", "schema1.table1"},
		{"some_schema", "table", "schema2.table1"},
		{"some_schema", "some_table", "schema2.table2"},
		// Add more tests
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
		{0, "      "},
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
		{0, 5, "     "},
		{0, 6, "      "},
		{1, 5, "    1"},
		{1, 6, "     1"},
		{1000, 6, "  1000"},
		{10000, 6, " 10000"},
		{100000, 6, "100000"},
		{1000000, 6, "1000000"},
	}

	for _, test := range tests {
		got := FormatCounter(test.counter, test.width)
		if got != test.expected {
			t.Errorf("FormatCounter(%v,%v) failed: expected: %q, got %q", test.counter, test.width, test.expected, got)
		}
	}
}
