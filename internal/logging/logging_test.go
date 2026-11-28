package logging

import (
	"context"
	"flag"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
)

// TestSetup checks which levels each flag combination enables, and which
// file, if any, it logs to.
func TestSetup(t *testing.T) {
	none := slog.Level(100) // above error: nothing enabled
	for _, tt := range []struct {
		args []string
		min  slog.Level // lowest level enabled
		file string     // file written, "" for none
	}{
		{nil, none, ""},
		{[]string{"--log"}, slog.LevelInfo, DefaultFile},
		{[]string{"--log=debug"}, slog.LevelDebug, DefaultFile},
		{[]string{"--log=WARN"}, slog.LevelWarn, DefaultFile}, // any case
		{[]string{"--log=error"}, slog.LevelError, DefaultFile},
		{[]string{"--log=true"}, slog.LevelInfo, DefaultFile},
		{[]string{"--log=false"}, none, ""},
		{[]string{"--log-file=p.log"}, slog.LevelInfo, "p.log"}, // alone: info
		{[]string{"--log=debug", "--log-file=p.log"}, slog.LevelDebug, "p.log"},
		{[]string{"--log=false", "--log-file=p.log"}, none, ""},                // false wins
		{[]string{"--log=bogus", "--log=debug"}, slog.LevelDebug, DefaultFile}, // last wins
	} {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			t.Chdir(t.TempDir())
			keepDefault(t)
			f := parse(t, tt.args...)
			if err := f.Setup(); err != nil {
				t.Fatal(err)
			}
			for _, l := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
				if got, want := slog.Default().Enabled(context.Background(), l), l >= tt.min; got != want {
					t.Errorf("%v enabled %v, want %v", l, got, want)
				}
			}
			slog.Error("probe")
			for _, name := range []string{DefaultFile, "p.log"} {
				b, err := os.ReadFile(name)
				if name != tt.file {
					if err == nil {
						t.Errorf("%s written, want not", name)
					}
					continue
				}
				if !strings.Contains(string(b), "level=ERROR msg=probe") {
					t.Errorf("%s holds %q, want the probe record", name, b)
				}
			}
		})
	}
}

func TestSetupErrors(t *testing.T) {
	for _, args := range [][]string{
		{"--log=bogus"},
		{"--log=INFO+2"}, // slog takes offsets: not here
		{"--log-file=" + t.TempDir() + "/no/such/dir/p.log"},
	} {
		keepDefault(t)
		if err := parse(t, args...).Setup(); err == nil {
			t.Errorf("%v: no error", args)
		}
	}
}

func TestHelpWanted(t *testing.T) {
	if !parse(t, "--log=help").HelpWanted() {
		t.Error("--log=help: HelpWanted false")
	}
	if parse(t, "--log").HelpWanted() {
		t.Error("--log: HelpWanted true")
	}
}

// TestUsage checks --help shows --log as a boolean flag and --log-file's
// value as a file with its default.
func TestUsage(t *testing.T) {
	fs := flag.NewFlagSet("ps-top", flag.ContinueOnError)
	AddFlags(fs)
	if name, _ := flag.UnquoteUsage(fs.Lookup("log")); name != "" {
		t.Errorf("--log value named %q, want none (a boolean flag)", name)
	}
	file := fs.Lookup("log-file")
	if name, _ := flag.UnquoteUsage(file); name != "file" {
		t.Errorf("--log-file value named %q, want file", name)
	}
	if file.DefValue != DefaultFile {
		t.Errorf("--log-file default %q, want %q", file.DefValue, DefaultFile)
	}
}

func parse(t *testing.T, args ...string) *Flags {
	t.Helper()
	fs := flag.NewFlagSet("ps-top", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	f := AddFlags(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}
	return f
}

// keepDefault restores slog's default logger when the test ends.
func keepDefault(t *testing.T) {
	d := slog.Default()
	t.Cleanup(func() { slog.SetDefault(d) })
}
