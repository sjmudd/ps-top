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
	data := []struct {
		input  uint64
		output string
	}{
		{0, ""},
		{1, "1 ps"},
		{1000, "   1.00 ns"},
		{1000000, "   1.00 us"},
		{1000000000, "   1.00 ms"},
		{1000000000000, "    1.00 s"},
		// add more values here
	}
	for i := range data {
		if FormatTime(data[i].input) != data[i].output {
			t.Errorf("FormatTime(%v) expected to be %v but actually was %v", data[i].input, data[i].output, FormatTime(data[i].input))
		}
	}
}
