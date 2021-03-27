package lib

import (
	"testing"
)

func TestProgName(t *testing.T) {
	const expected = "lib.test"
	if ProgName != expected {
		t.Errorf("ProgName expected to be %v but actually was %v", expected, ProgName)
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
