package anonymise

import (
	"sync"
	"testing"
)

func TestAnonymise(t *testing.T) {
	a := New()
	tests := []struct{ name, group, want string }{
		{"shop", "schema", "schema1"},
		{"crm", "schema", "schema2"},
		{"shop", "schema", "schema1"}, // same name again
		{"orders", "table", "table1"}, // groups are numbered on their own
		{"shop", "table", "table2"},   // a name in another group is another name
		{"", "user", ""},              // empty stays empty
		{"app", "user", "user1"},      // and takes no number
		{"orders", "table", "table1"},
	}
	for _, tt := range tests {
		if got := a.Anonymise(tt.name, tt.group); got != tt.want {
			t.Errorf("Anonymise(%q, %q) = %q, want %q", tt.name, tt.group, got, tt.want)
		}
	}
}

func TestExclude(t *testing.T) {
	a := New()
	a.Exclude("mysql", "schema")
	tests := []struct{ name, group, want string }{
		{"mysql", "schema", "mysql"},  // excluded
		{"shop", "schema", "schema1"}, // the excluded name took no number
		{"mysql", "table", "table1"},  // excluded only in its group
		{"mysql", "schema", "mysql"},
	}
	for _, tt := range tests {
		if got := a.Anonymise(tt.name, tt.group); got != tt.want {
			t.Errorf("Anonymise(%q, %q) = %q, want %q", tt.name, tt.group, got, tt.want)
		}
	}
	if !a.Excluded("mysql", "schema") || a.Excluded("mysql", "table") || a.Excluded("shop", "schema") {
		t.Error("Excluded: want true only for mysql as a schema")
	}
	// Excluded after it was anonymised: the name is now left as it is.
	a.Exclude("shop", "schema")
	if got := a.Anonymise("shop", "schema"); got != "shop" {
		t.Errorf("after Exclude: got %q, want shop", got)
	}
}

func TestAnonymiseConcurrent(t *testing.T) {
	a := New()
	var wg sync.WaitGroup
	got := make([]string, 50)
	for i := range got {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got[i] = a.Anonymise("shop", "schema")
		}()
	}
	wg.Wait()
	for i, g := range got {
		if g != "schema1" {
			t.Errorf("call %d: got %q, want schema1", i, g)
		}
	}
}
