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
		// add more values here
	}
	for _, test := range tests {
		got := FormatTime(test.picoseconds)
		if got != test.expected {
			t.Errorf("FormatTime(%v) failed: expected: %q, got %q", test.picoseconds, test.expected, got)
		}
	}
}
