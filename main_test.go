package main

import (
	"flag"
	"io"
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want time.Duration
	}{
		{"3", 3 * time.Second}, // a whole number is seconds
		{"3s", 3 * time.Second},
		{"0", 0},
		{"60", time.Minute},
		{"1m", time.Minute},
		{"500ms", 500 * time.Millisecond},
		{"-3s", -3 * time.Second}, // as time.ParseDuration: the app checks the sign
	} {
		if got, err := parseDuration(tt.in); err != nil || got != tt.want {
			t.Errorf("parseDuration(%q) = %v, %v, want %v", tt.in, got, err, tt.want)
		}
	}
	for _, in := range []string{"", "3x", "1.5", "-3", "s", "9223372037"} {
		if got, err := parseDuration(in); err == nil {
			t.Errorf("parseDuration(%q) = %v, want an error", in, got)
		}
	}
}

// TestDurationVar checks a flag set with durationVar takes seconds and
// units, keeps its default, and shows it as a duration.
func TestDurationVar(t *testing.T) {
	fs := flag.CommandLine
	defer func() { flag.CommandLine = fs }()
	flag.CommandLine = flag.NewFlagSet("ps-top", flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)

	var interval, window time.Duration
	durationVar(&interval, "interval", time.Second, "")
	durationVar(&window, "window", 10*time.Second, "")
	if err := flag.CommandLine.Parse([]string{"--interval", "3"}); err != nil {
		t.Fatal(err)
	}
	if interval != 3*time.Second || window != 10*time.Second {
		t.Errorf("interval %v, window %v, want 3s and 10s", interval, window)
	}
	if got := flag.CommandLine.Lookup("window").DefValue; got != "10s" {
		t.Errorf("window default shown as %q, want 10s", got)
	}
	if name, _ := flag.UnquoteUsage(flag.CommandLine.Lookup("window")); name != "duration" {
		t.Errorf("--help names the value %q, want duration", name)
	}
	if err := flag.CommandLine.Parse([]string{"--interval=3x"}); err == nil {
		t.Error("--interval=3x: no error")
	}
}
