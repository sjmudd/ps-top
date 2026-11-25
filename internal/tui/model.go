package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/presenter"
)

// tickMsg asks for a redraw: one each Config.Interval.
type tickMsg struct{}

// renderedMsg is a render's result: the table of render seq, or its error.
type renderedMsg struct {
	seq   int
	table presenter.Table
	err   error
}

// windowMsg is the collector's answer to a window change to n intervals.
type windowMsg struct {
	n   int
	err error
}

// resetMsg is the collector's answer to a reset.
type resetMsg struct{ err error }

// windowSteps are the windows w and W step through, those that are a whole
// multiple of the collect interval.
var windowSteps = []time.Duration{5 * time.Second, 10 * time.Second, 30 * time.Second, 60 * time.Second}

// model is the TUI's state. Update changes it, View draws it. Neither does
// I/O: renders run as commands.
type model struct {
	ctx   context.Context
	cfg   Config
	now   func() time.Time
	theme theme

	width, height int
	view          int  // index into cfg.Views
	help          bool // the help screen is shown
	first         bool // no table shown yet: render cfg.First

	interval  time.Duration // time between redraws
	n         int           // the window, in collect intervals
	flipped   bool          // views show their other style (the r key)
	anonymise bool          // names are anonymised (the a key)

	table presenter.Table // the latest table shown
	have  bool            // table is set

	seq   int  // the latest render's sequence number
	busy  bool // a render is running
	dirty bool // a render was asked for while one ran: render again when it ends

	message      string
	messageUntil time.Time

	err error // a render failed: Run returns it
}

func newModel(ctx context.Context, cfg Config, now func() time.Time) *model {
	m := &model{ctx: ctx, cfg: cfg, now: now, theme: oldTheme(), first: true,
		interval: cfg.Interval, n: cfg.Options.N, anonymise: cfg.Options.Anonymiser != nil}
	if cfg.Message != "" {
		m.setMessage(cfg.Message)
	}
	return m
}

// Init starts the first render and the redraw ticker.
func (m *model) Init() tea.Cmd { return tea.Batch(m.render(), m.tick()) }

func (m *model) tick() tea.Cmd {
	return tea.Tick(m.interval, func(time.Time) tea.Msg { return tickMsg{} })
}

// rowLimit returns how many rows fit: the screen less the top line, title,
// headings, totals and menu.
func (m *model) rowLimit() int { return max(m.height-5, 1) }

// render returns a command rendering the current view, or nil if a render
// is running: it is then rendered again when that one ends. A table from an
// earlier render than the latest is dropped.
func (m *model) render() tea.Cmd {
	m.seq++
	if m.busy {
		m.dirty = true
		return nil
	}
	m.busy = true
	v, src, ctx, seq := m.current(), m.cfg.Source, m.ctx, m.seq
	o := m.cfg.Options
	o.N, o.Limit, o.Width, o.TotalsOnly = m.n, m.rowLimit(), 0, false
	o.Anonymiser = nil
	if m.anonymise {
		o.Anonymiser = m.cfg.Anonymiser
	}
	return func() tea.Msg {
		t, err := v.Table(ctx, src, o)
		return renderedMsg{seq: seq, table: t, err: err}
	}
}

// current returns the view to render: the first view until a table is
// shown, else the view, in its other style if flipped and it has one.
func (m *model) current() presenter.Renderer {
	if m.first {
		return m.cfg.First[m.view]
	}
	v := m.cfg.Views[m.view]
	if m.flipped {
		if other, ok := otherStyle(v); ok {
			return other
		}
	}
	return v
}

// otherStyle returns v in its other style: current values for a difference,
// a difference for current values. False if v's metric has only one.
func otherStyle(v presenter.Renderer) (presenter.Renderer, bool) {
	t := metric.CurrentValue
	if v.Style() == metric.CurrentValue.Name() {
		t = metric.DifferenceValues
	}
	if !v.Accepts(t) {
		return v, false
	}
	other, err := v.WithStyle(t)
	return other, err == nil
}

// setMessage shows text on the screen for cfg.MessageTime.
func (m *model) setMessage(text string) {
	m.message = text
	m.messageUntil = m.now().Add(m.cfg.MessageTime)
}

// Update handles one message.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		before := m.rowLimit()
		m.width, m.height = msg.Width, msg.Height
		if m.rowLimit() != before {
			return m, m.render()
		}
	case tea.KeyPressMsg:
		return m.key(msg.String())
	case tea.ResumeMsg:
		return m, m.render()
	case tickMsg:
		return m, tea.Batch(m.render(), m.tick())
	case renderedMsg:
		return m.rendered(msg)
	case windowMsg:
		if msg.err != nil {
			return m, m.failed("window", msg.err)
		}
		m.n = msg.n
		m.setMessage("window " + secs(m.windowTime(m.n)))
		return m, m.render()
	case resetMsg:
		if msg.err != nil {
			return m, m.failed("reset", msg.err)
		}
		m.setMessage("reset: windows start again from now")
		return m, m.render()
	}
	return m, nil
}

// failed shows that a change to the collector failed, unless stopping.
func (m *model) failed(what string, err error) tea.Cmd {
	if m.ctx.Err() == nil {
		m.setMessage(what + ": " + err.Error())
	}
	return nil
}

// windowTime returns the time a window of n collect intervals covers.
func (m *model) windowTime(n int) time.Duration {
	return time.Duration(n) * m.cfg.CollectInterval
}

// stepWindow returns a command changing the window to the next step down
// (or up), or nil with a message if there is none.
func (m *model) stepWindow(up bool) tea.Cmd {
	if m.cfg.Control == nil || m.cfg.CollectInterval <= 0 {
		m.setMessage("the window cannot be changed")
		return nil
	}
	now := m.windowTime(m.n)
	next := time.Duration(0)
	for _, s := range windowSteps {
		if s%m.cfg.CollectInterval != 0 {
			continue
		}
		if up && s > now && (next == 0 || s < next) || !up && s < now && s > next {
			next = s
		}
	}
	if next == 0 {
		end := "shortest"
		if up {
			end = "longest"
		}
		m.setMessage("window " + secs(now) + ": the " + end)
		return nil
	}
	ctl, ctx, n := m.cfg.Control, m.ctx, int(next/m.cfg.CollectInterval)
	return func() tea.Msg { return windowMsg{n: n, err: ctl.SetWindow(ctx, n)} }
}

// key handles a key press.
func (m *model) key(k string) (tea.Model, tea.Cmd) {
	act, ok := lookup(k)
	if !ok {
		return m, nil
	}
	switch act {
	case actQuit:
		return m, tea.Quit
	case actSuspend:
		return m, tea.Suspend
	case actHelp:
		m.help = !m.help
	case actNext:
		m.view = (m.view + 1) % len(m.cfg.Views)
		m.help = false
		return m, m.render()
	case actPrev:
		m.view = (m.view + len(m.cfg.Views) - 1) % len(m.cfg.Views)
		m.help = false
		return m, m.render()
	case actIntervalUp:
		m.interval += time.Second
		m.setMessage("interval " + secs(m.interval))
	case actIntervalDown:
		if m.interval-time.Second < max(m.cfg.CollectInterval, time.Second) {
			m.setMessage("interval " + secs(m.interval) + ": the shortest")
			break
		}
		m.interval -= time.Second
		m.setMessage("interval " + secs(m.interval))
	case actWindowDown:
		return m, m.stepWindow(false)
	case actWindowUp:
		return m, m.stepWindow(true)
	case actStyle:
		m.flipped = !m.flipped
		if _, ok := otherStyle(m.cfg.Views[m.view]); !ok {
			m.setMessage("this view has one style: " + m.cfg.Views[m.view].Style())
		}
		return m, m.render()
	case actAnonymise:
		m.anonymise = !m.anonymise
		state := "off"
		if m.anonymise {
			state = "on"
		}
		m.setMessage("anonymise " + state)
		return m, m.render()
	case actReset:
		if m.cfg.Control == nil {
			m.setMessage("the collector cannot be reset")
			break
		}
		ctl, ctx := m.cfg.Control, m.ctx
		return m, func() tea.Msg { return resetMsg{err: ctl.Reset(ctx)} }
	}
	return m, nil
}

// rendered keeps a render's table if it is the latest, and starts another
// render if one was asked for meanwhile.
func (m *model) rendered(msg renderedMsg) (tea.Model, tea.Cmd) {
	m.busy = false
	if msg.seq != m.seq {
		if m.dirty {
			m.dirty = false
			return m, m.render()
		}
		return m, nil
	}
	if msg.err != nil {
		if m.ctx.Err() != nil {
			return m, nil // stopping: the render was cut short
		}
		m.err = msg.err
		return m, tea.Quit
	}
	m.table, m.have = msg.table, true
	if !msg.table.Waiting {
		m.first = false
	}
	return m, nil
}
