package munge

import (
	"os"
	"path/filepath"
	"testing"
)

// write writes text to a file in a test directory and returns its path.
func write(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), FileName)
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadAndApply(t *testing.T) {
	path := write(t, `
[other]
x = y

[munge]
_[0-9]{8}$ = _YYYYMMDD
^archive_[a-z]+\. = archive.
([ = never used: does not compile
`)
	rules, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 {
		t.Fatalf("%d rules, want 2 (one skipped)", len(rules))
	}
	for name, want := range map[string]string{
		"shop.orders_20260101":    "shop.orders_YYYYMMDD",
		"shop.orders_2026010":     "shop.orders_2026010",
		"archive_eu.orders":       "archive.orders",
		"archive_eu.log_20260101": "archive.log_YYYYMMDD",
		"x":                       "x",
	} {
		if got := rules.Apply(name); got != want {
			t.Errorf("Apply(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestLoadNoRules(t *testing.T) {
	for _, path := range []string{"", filepath.Join(t.TempDir(), "missing")} {
		rules, err := Load(path)
		if err != nil || rules != nil {
			t.Errorf("Load(%q) = %v, %v: want no rules, no error", path, rules, err)
		}
	}
	if got := Rules(nil).Apply("shop.orders"); got != "shop.orders" {
		t.Errorf("nil rules: Apply = %q", got)
	}
}

func TestLoadSyntaxError(t *testing.T) {
	if _, err := Load(write(t, "[munge\n")); err == nil {
		t.Error("no error for a file that does not parse")
	}
}
