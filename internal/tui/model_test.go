package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"github.com/sjmudd/ps-top/v2/internal/metric"
	"github.com/sjmudd/ps-top/v2/internal/presenter"
	"github.com/sjmudd/ps-top/v2/internal/registry"
)

// TestPartLines checks a combined view's screen: the top parts from the top,
// the last at the foot, and the top cut first when it does not fit.
func TestPartLines(t *testing.T) {
	part := func(title string) presenter.Table {
		return presenter.Table{Title: title, Waiting: true} // the title line only
	}
	parts := []presenter.Table{part("a"), part("b"), part("c")}
	got := partLines(parts, 7)
	want := []string{"a", "", "b", "", "", "", "c"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("7 lines: %q, want %q", got, want)
	}
	got = partLines(parts, 3)
	want = []string{"a", "", "c"} // b is cut
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("3 lines: %q, want %q", got, want)
	}
}

// fakeView is a view returning a fixed table, recording the options and
// style of each render. Its style is "difference" unless set, and it
// accepts both styles unless one is set as its only one.
type fakeView struct {
	name   string
	table  presenter.Table
	err    error
	got    *[]presenter.Options
	styles *[]string // the style of each render
	style  string
	only   bool // accepts its style only
}

func (f fakeView) ViewName() string                                 { return f.name }
func (f fakeView) Metric() string                                   { return f.name }
func (f fakeView) WithStats(*registry.Presenter) presenter.Renderer { return f }

func (f fakeView) Style() string {
	if f.style == "" {
		return metric.DifferenceValues.Name()
	}
	return f.style
}

func (f fakeView) Accepts(t metric.MetricType) bool { return !f.only || t.Name() == f.Style() }

func (f fakeView) WithStyle(t metric.MetricType) (presenter.Renderer, error) {
	f.style = t.Name()
	return f, nil
}

func (f fakeView) Render(context.Context, presenter.Source, presenter.Options) ([]string, error) {
	return f.table.Lines(0), f.err
}

func (f fakeView) Table(_ context.Context, _ presenter.Source, o presenter.Options) (presenter.Table, error) {
	if f.got != nil {
		*f.got = append(*f.got, o)
	}
	if f.styles != nil {
		*f.styles = append(*f.styles, f.Style())
	}
	return f.table, f.err
}

// fakeControl records the collector changes asked for, failing with err.
type fakeControl struct {
	windows []int
	resets  int
	err     error
}

func (c *fakeControl) SetWindow(_ context.Context, n int) error {
	c.windows = append(c.windows, n)
	return c.err
}

func (c *fakeControl) Reset(context.Context) error {
	c.resets++
	return c.err
}

// sample is a table of two rows, with a "|" before the names.
func sample(title string) presenter.Table {
	group := func(t string) presenter.TableGroup { return presenter.TableGroup{{Text: t}} }
	row := func(v, k string) []presenter.TableGroup { return []presenter.TableGroup{group(v), group(k)} }
	return presenter.Table{
		Title:    title,
		Style:    "last 10s",
		Covers:   10 * time.Second,
		Headings: row("Latency", "Name"),
		Rows:     [][]presenter.TableGroup{row("   2 ms", "shop.a"), row("   1 ms", "shop.b")},
		Totals:   row("   3 ms", "Totals"),
	}
}

var t0 = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// plain is a theme without colours, so screens compare as text.
func plain() theme {
	s := lipgloss.NewStyle()
	return theme{s, s, s, s, s, s, s, s, s}
}

// newTest returns a model of the views, w by h, at time t0, with plain
// styles. Views are their own first views.
func newTest(w, h int, views ...presenter.Renderer) *model {
	cfg := Config{
		Program: "ps-top", Version: "2.0.0-dev",
		Server: ServerInfo{Host: "db1", Port: "3306", Version: "8.4.11", Started: t0.Add(-93784 * time.Second)},
		Views:  views, First: views, Interval: time.Second, MessageTime: 3 * time.Second,
	}
	m := newModel(context.Background(), cfg, func() time.Time { return t0 })
	m.theme = plain()
	m.width, m.height = w, h
	return m
}

// run runs cmd and hands its message to Update, as Bubble Tea would.
func run(t *testing.T, m *model, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		t.Fatal("no command")
	}
	m.Update(cmd())
}

func screenLines(m *model) []string { return strings.Split(m.screen(), "\n") }

// TestScreen checks the areas: top line, title, headings, the rows then
// empty rows showing the "|", totals on the second-last line, and the menu,
// each padded to the width.
func TestScreen(t *testing.T) {
	m := newTest(110, 10, fakeView{name: "a", table: sample("Table I/O Latency 2 rows")})
	m.cfg.CollectInterval, m.n = time.Second, 10
	run(t, m, m.render())
	lines := screenLines(m)
	if len(lines) != 10 {
		t.Fatalf("%d lines, want 10:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	for i, l := range lines {
		if runes(l) != 110 {
			t.Errorf("line %d is %d wide, want 110: %q", i+1, runes(l), l)
		}
	}
	want := map[int]string{
		1: "Table I/O Latency 2 rows",
		2: "Latency|Name",
		3: "   2 ms|shop.a",
		4: "   1 ms|shop.b",
		5: "       |",
		6: "       |",
		7: "       |",
		8: "   3 ms|Totals",
		9: "[+-] Interval  [<] Prev  [>] Next  [w/W] Window  [r] Style  [a] Anonymise  [z] Reset  [h]elp  [q]uit",
	}
	for i, w := range want {
		if got := strings.TrimRight(lines[i], " "); got != w {
			t.Errorf("line %d = %q, want %q", i+1, got, w)
		}
	}
	top := lines[0]
	if !strings.HasPrefix(top, "ps-top 2.0.0-dev -  3:04:05 db1:3306 / 8.4.11, up 1d 2h 3m 4s") || !strings.HasSuffix(top, "[collect: 1s, window: 10s]") {
		t.Errorf("top line %q", top)
	}
}

// TestCoverText checks the top line's right side: last is shown only when it
// is not the window.
func TestCoverText(t *testing.T) {
	m := newTest(110, 10, fakeView{name: "a"})
	for _, tt := range []struct {
		collect time.Duration
		n       int
		style   string
		covers  time.Duration
		want    string
	}{
		{time.Second, 30, "last 30s", 30*time.Second + 2*time.Millisecond, "collect: 1s, window: 30s"},
		{time.Second, 30, "last 5s", 5 * time.Second, "collect: 1s, window: 30s, last: 5s"}, // the window filling
		{time.Second, 30, "current", 0, "collect: 1s, current"},
		{3 * time.Second, 3, "last 9s", 9 * time.Second, "collect: 3s, window: 9s"}, // --window=10s
	} {
		m.cfg.CollectInterval, m.n = tt.collect, tt.n
		m.table = presenter.Table{Style: tt.style, Covers: tt.covers}
		if got := m.coverText(); got != tt.want {
			t.Errorf("collect %v, n %d, %q: %q, want %q", tt.collect, tt.n, tt.style, got, tt.want)
		}
	}
}

// TestRowLimit checks a render asks for the rows that fit, and a resize
// that changes them renders again.
func TestRowLimit(t *testing.T) {
	var got []presenter.Options
	m := newTest(60, 10, fakeView{name: "a", table: sample("t"), got: &got})
	run(t, m, m.render())
	_, cmd := m.Update(tea.WindowSizeMsg{Width: 60, Height: 8})
	run(t, m, cmd)
	if _, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 8}); cmd != nil {
		t.Error("width change only: rendered again")
	}
	if len(got) != 2 || got[0].Limit != 5 || got[1].Limit != 3 {
		t.Errorf("renders %+v, want limits 5 then 3", got)
	}
}

// TestKeys checks each key's action, and a key press reaching it through
// Update.
func TestKeys(t *testing.T) {
	m := newTest(60, 30, fakeView{name: "a"}, fakeView{name: "b"}, fakeView{name: "c"})
	run(t, m, m.render())
	steps := []struct {
		key  string
		view int
	}{{"tab", 1}, {"right", 2}, {"tab", 0}, {"shift+tab", 2}, {"left", 1}}
	for _, s := range steps {
		_, cmd := m.key(s.key)
		if m.view != s.view || cmd == nil {
			t.Fatalf("%s: view %d, render %v, want view %d and a render", s.key, m.view, cmd != nil, s.view)
		}
		run(t, m, cmd)
	}
	m.key("h")
	if !m.help || !strings.Contains(m.screen(), "q, Esc, Ctrl-C") || strings.TrimSpace(screenLines(m)[1]) != "Help" {
		t.Errorf("h: help %v, screen:\n%s", m.help, m.screen())
	}
	m.key("?")
	if m.help {
		t.Error("?: help still shown")
	}
	for _, k := range []string{"q", "esc", "ctrl+c"} {
		if _, cmd := m.key(k); cmd == nil || cmd() != tea.Quit() {
			t.Errorf("%s: does not quit", k)
		}
	}
	if _, cmd := m.key("ctrl+z"); cmd == nil || cmd() != tea.Suspend() {
		t.Error("ctrl+z: does not suspend")
	}
	if _, cmd := m.key("x"); cmd != nil {
		t.Error("x: does something")
	}
	if _, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"}); cmd == nil || cmd() != tea.Quit() {
		t.Error("key press q through Update: does not quit")
	}
}

// TestRenderSequence checks a render asked for while one runs follows it,
// and the earlier table is dropped.
func TestRenderSequence(t *testing.T) {
	m := newTest(60, 10, fakeView{name: "a", table: sample("first")}, fakeView{name: "b", table: sample("second")})
	cmd := m.render()
	if _, again := m.key("tab"); again != nil {
		t.Fatal("render started while one runs")
	}
	_, next := m.Update(cmd()) // the first render ends: dropped, the next starts
	if m.have || next == nil {
		t.Fatalf("stale table kept %v, next render %v", m.have, next != nil)
	}
	run(t, m, next)
	if m.table.Title != "second" {
		t.Errorf("table %q, want second", m.table.Title)
	}
}

// TestFirstViews checks the first screen uses the first views, and later
// ones the views, once a table is shown.
func TestFirstViews(t *testing.T) {
	m := newTest(60, 10, fakeView{name: "a", table: sample("difference")})
	m.cfg.First = []presenter.Renderer{fakeView{name: "a", table: sample("current")}}
	run(t, m, m.render())
	run(t, m, m.render())
	if m.table.Title != "difference" || m.first {
		t.Errorf("second render %q, first %v: want difference", m.table.Title, m.first)
	}

	w := newTest(60, 10, fakeView{name: "a", table: presenter.Table{Title: "a (waiting for data)", Waiting: true}})
	run(t, w, w.render())
	if !w.first || strings.TrimSpace(screenLines(w)[1]) != "a (waiting for data)" {
		t.Errorf("waiting: first %v, screen\n%s", w.first, w.screen())
	}
}

// TestMessage checks a message is shown centred on the middle row for the
// message time, then no more.
func TestMessage(t *testing.T) {
	m := newTest(60, 10, fakeView{name: "a", table: sample("t")})
	run(t, m, m.render())
	m.setMessage("window 30s")
	row := screenLines(m)[3+(10-5)/2]
	if strings.TrimSpace(row) != "window 30s" || !strings.HasPrefix(row, strings.Repeat(" ", 20)) {
		t.Errorf("message row %q", row)
	}
	now := t0.Add(3 * time.Second)
	m.now = func() time.Time { return now }
	if strings.Contains(m.screen(), "window 30s") {
		t.Error("message shown after its time")
	}
}

// TestRenderError checks a failed render quits with its error, unless the
// program is stopping.
func TestRenderError(t *testing.T) {
	fail := errors.New("collector gone")
	m := newTest(60, 10, fakeView{name: "a", err: fail})
	_, cmd := m.Update(m.render()())
	if !errors.Is(m.err, fail) || cmd == nil || cmd() != tea.Quit() {
		t.Errorf("err %v, quit %v", m.err, cmd != nil)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := newTest(60, 10, fakeView{name: "a", err: fail})
	s.ctx = ctx
	s.Update(s.render()())
	if s.err != nil {
		t.Errorf("stopping: err %v, want none", s.err)
	}
}

func TestTooSmall(t *testing.T) {
	if got := newTest(19, 10, fakeView{name: "a"}).screen(); got != "terminal too small: 19 by 10" {
		t.Errorf("got %q", got)
	}
}

func TestUptime(t *testing.T) {
	for _, tt := range []struct {
		secs int
		want string
	}{{0, "0s"}, {36, "36s"}, {60, "1m 0s"}, {3600, "1h 0m 0s"}, {2876134, "33d 6h 55m 34s"}} {
		if got := uptime(time.Duration(tt.secs) * time.Second); got != tt.want {
			t.Errorf("uptime(%ds) = %q, want %q", tt.secs, got, tt.want)
		}
	}
	if got := clock(time.Date(2026, 1, 1, 0, 43, 26, 0, time.UTC)); got != " 0:43:26" {
		t.Errorf("clock = %q", got)
	}
}
