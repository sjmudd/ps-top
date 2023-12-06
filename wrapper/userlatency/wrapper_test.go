package userlatency

import (
	"testing"
)

func TestFormatSeconds(t *testing.T) {
	data := []struct {
		input  uint64
		output string
	}{
		{0, ""},
		{1, "1s"},
		{10, "10s"},
		{60, "1m 0s"},
		{70, "1m 10s"},
		{3599, "59m 59s"},
		{3600, "1h 0m 0s"},
		{3601, "1h 0m 1s"},
		{36001, "10h 0m 1s"},
		{36010, "10h 0m 10s"}, // max width
		{36600, "10h 10m 0s"}, // max width
		{36610, "10h 10m"},    // truncate due to > 10 characters
		{86399, "23h 59m"},    // truncate due to > 10 characters
		{86400, "1d 0h 0m"},
		{86401, "1d 0h 0m"},
		{86460, "1d 0h 1m"},
	}
	for i := range data {
		if formatSeconds(data[i].input) != data[i].output {
			t.Errorf("formatSeconds(%v) expected: %v, got: %v", data[i].input, data[i].output, formatSeconds(data[i].input))
		}
	}
}
