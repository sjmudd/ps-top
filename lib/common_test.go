package lib

import (
	"testing"
)

func TestMyName(t *testing.T) {
	const expected = "lib.test"
	if MyName() != expected {
		t.Errorf("MyName() expected to be %v but actually was %v", expected, MyName())
	}
}

func TestFormatTime(t *testing.T) {
	type stuff struct {
		input  uint64
		output string
	}
	testData := []stuff{
		{0, ""},
		{1, "1 ps"},
		{1000, "   1.00 ns"},
		{1000000, "   1.00 us"},
		{1000000000, "   1.00 ms"},
		{1000000000000, "    1.00 s"},
		// add more values here
	}
	for i := range testData {
		if FormatTime(testData[i].input) != testData[i].output {
			t.Errorf("FormatTime(%v) expected to be %v but actually was %v", testData[i].input, testData[i].output, FormatTime(testData[i].input))
		}
	}
}
