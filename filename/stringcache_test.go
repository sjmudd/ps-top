package filename

import (
	"testing"
)

func TestGetAndPut(t *testing.T) {
	// check not found in cache
	k, v := "key", "value"
	if _, err := cache.get(k); err == nil {
		t.Errorf("cache.get(%q) gave no error: expected %+v", k, err)
	}
	// add a value
	v2 := cache.put(k, v)
	if v2 != v {
		t.Errorf("cache.put(%q,%q) returned %q: expected: %q", k, v, v2, v)
	}
	// check it's readable
	v3, err := cache.get(k)
	if err != nil {
		t.Errorf("cache.get(%q) gave an error: %v. not expecting one", k, err)
	}
	if v3 != v {
		t.Errorf("cache.get(%q) returned %q: expected: %q", k, v3, v)
	}
}
