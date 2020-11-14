// Package file_io contains the routines for
// managing the file_summary_by_instance table.
package file_io

import (
	"testing"
)

func TestAdd(t *testing.T) {
	var tests = []struct {
		val1 Row
		val2 Row
		sum  Row
	}{
		{
			Row{"name1", 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			Row{"any__", 101, 102, 103, 104, 105, 106, 107, 108, 109, 110},
			Row{"name1", 102, 104, 106, 108, 110, 112, 114, 116, 118, 120}},
	}

	for _, test := range tests {
		result := add(test.val1, test.val2)
		if result != test.sum {
			t.Errorf("r(%v).add(%v): expected %v, actual %v", test.val1, test.val2, test.sum, result)
		}
		if result.Name != test.val1.Name {
			t.Errorf("r(%v).add(%v): name has changed from '%s' to '%s'", test.val1, test.val2, test.val1.Name, result.Name)
		}
	}
}

func TestSubtract(t *testing.T) {
	var tests = []struct {
		val1 Row
		val2 Row
		diff Row
	}{
		{
			Row{"name1", 102, 104, 106, 108, 110, 112, 114, 116, 118, 120},
			Row{"any__", 101, 102, 103, 104, 105, 106, 107, 108, 109, 110},
			Row{"name1", 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
	}

	for _, test := range tests {
		result := subtract(test.val1, test.val2)
		if result != test.diff {
			t.Errorf("r(%v).subtract(%v): expected %v, actual %v", test.val1, test.val2, test.diff, result)
		}
		if result.Name != test.val1.Name {
			t.Errorf("r(%v).add(%v): name has changed from '%s' to '%s'", test.val1, test.val2, test.val1.Name, result.Name)
		}
	}
}
