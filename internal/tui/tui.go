// Package tui is ps-top's terminal interface: a Bubble Tea program that
// draws one view at a time, full screen, and redraws it every interval.
//
// The screen, from the top, in a terminal of height H:
//
//	line 1          top line: program, version, time, server, uptime, and what the values cover
//	line 2          the view's title
//	line 3          headings
//	lines 4 to H-2  rows: the first H-5, then blank lines
//	line H-1        totals
//	line H          menu: the keys
//
// Help replaces lines 2 to H-1. A message is shown on the middle line of the
// rows for a while (Config.MessageTime).
//
// The model never waits on the collector: a render is a Bubble Tea command,
// run in its own goroutine, which asks the view for its presenter.Table. At
// most one render runs at a time, and each has a sequence number, so a table
// from before a key press is not shown after it.
package tui

import (
	"context"
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/sjmudd/ps-top/v2/internal/anonymise"
	"github.com/sjmudd/ps-top/v2/internal/presenter"
)

// ServerInfo is what the top line shows about the server, e.g.
// "db1:3306 / 8.4.11, up 2d 3h 4m 5s".
type ServerInfo struct {
	Host    string    // bind_address if it names one address, else hostname
	Port    string    // e.g. "3306"
	Version string    // e.g. "8.4.11"
	Started time.Time // when the server started: the uptime is the time since
}

// Control changes the collector while running: collector.Collector.
type Control interface {
	SetWindow(ctx context.Context, n int) error
	Reset(ctx context.Context) error
}

// Config is what the TUI shows and how.
type Config struct {
	Program string // the program's name, as run, e.g. "ps-top"
	Version string // ps-top's version
	Server  ServerInfo
	// Views are the views, in the order Tab steps through them, and First
	// the same views for the first screen: values up to now, as vmstat's
	// first line.
	Views, First []presenter.Renderer
	Source       presenter.Source
	Control      Control // for the window (w, W) and reset (z) keys; nil: they say so
	// Options are the render options at start: N the window in collect
	// intervals, Anonymiser set if anonymising is on. The TUI sets Limit to
	// the rows that fit, and ignores Width and TotalsOnly.
	Options presenter.Options
	// Anonymiser is used while anonymising is on (the a key): one for the
	// whole run, so names keep their numbers when toggled back.
	Anonymiser      *anonymise.Anonymiser
	CollectInterval time.Duration // the window is a whole number of it, and the interval at least it
	Interval        time.Duration // time between redraws at start
	MessageTime     time.Duration // how long a message is shown
	Message         string        // a message to show at start, e.g. metrics skipped
}

// Run runs the TUI until the user quits, or ctx is done (a signal, or the
// collector stopping): both return nil. It returns a render's error.
func Run(ctx context.Context, cfg Config) error {
	if len(cfg.Views) == 0 || len(cfg.First) != len(cfg.Views) {
		return errors.New("tui: no views, or First and Views differ in number")
	}
	m := newModel(ctx, cfg, time.Now)
	final, err := tea.NewProgram(m, tea.WithContext(ctx), tea.WithoutSignalHandler()).Run()
	if errors.Is(err, tea.ErrProgramKilled) {
		err = nil // stopped from outside
	}
	if err != nil {
		return err
	}
	if fm, ok := final.(*model); ok {
		return fm.err
	}
	return nil
}
