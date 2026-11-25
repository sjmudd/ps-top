package tui

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/sjmudd/ps-top/internal/anonymise"
	"github.com/sjmudd/ps-top/internal/presenter"
)

// keyCmd presses k and runs its command, if any, through Update.
func keyCmd(t *testing.T, m *model, k string) {
	t.Helper()
	if _, cmd := m.key(k); cmd != nil {
		m.Update(cmd())
	}
}

func TestIntervalKeys(t *testing.T) {
	m := newTest(60, 10, fakeView{name: "a"})
	m.cfg.CollectInterval, m.interval = time.Second, 2*time.Second
	for _, s := range []struct {
		key      string
		interval time.Duration
		message  string
	}{
		{"+", 3 * time.Second, "interval 3s"},
		{"-", 2 * time.Second, "interval 2s"},
		{"-", time.Second, "interval 1s"},
		{"-", time.Second, "interval 1s: the shortest"},
	} {
		m.key(s.key)
		if m.interval != s.interval || m.message != s.message {
			t.Errorf("%s: interval %v, message %q, want %v and %q", s.key, m.interval, m.message, s.interval, s.message)
		}
	}
}

func TestWindowKeys(t *testing.T) {
	ctl := &fakeControl{}
	m := newTest(60, 10, fakeView{name: "a"})
	m.cfg.Control, m.cfg.CollectInterval, m.n = ctl, time.Second, 10
	for _, s := range []struct {
		key     string
		n       int
		message string
	}{
		{"w", 5, "window 5s"},
		{"w", 5, "window 5s: the shortest"},
		{"W", 10, "window 10s"},
		{"W", 30, "window 30s"},
		{"W", 60, "window 60s"},
		{"W", 60, "window 60s: the longest"},
	} {
		keyCmd(t, m, s.key)
		if m.n != s.n || m.message != s.message {
			t.Errorf("%s: n %d, message %q, want %d and %q", s.key, m.n, m.message, s.n, s.message)
		}
	}
	if want := []int{5, 10, 30, 60}; !slices.Equal(ctl.windows, want) {
		t.Errorf("SetWindow calls %v, want %v", ctl.windows, want)
	}

	// A collect interval of 2 s: 5 s is not a whole multiple, so 10 s is the
	// shortest.
	m.cfg.CollectInterval, m.n = 2*time.Second, 5
	keyCmd(t, m, "w")
	if m.n != 5 || m.message != "window 10s: the shortest" {
		t.Errorf("2 s collect: n %d, message %q", m.n, m.message)
	}

	ctl.err = errors.New("collector stopped")
	keyCmd(t, m, "W")
	if m.n != 5 || m.message != "window: collector stopped" {
		t.Errorf("failed change: n %d, message %q", m.n, m.message)
	}

	m.cfg.Control = nil
	keyCmd(t, m, "W")
	if m.message != "the window cannot be changed" {
		t.Errorf("no control: message %q", m.message)
	}
}

func TestResetKey(t *testing.T) {
	ctl := &fakeControl{}
	m := newTest(60, 10, fakeView{name: "a"})
	m.cfg.Control = ctl
	keyCmd(t, m, "z")
	if ctl.resets != 1 || m.message != "reset: windows start again from now" {
		t.Errorf("resets %d, message %q", ctl.resets, m.message)
	}
}

// TestStyleKey checks r flips each view to its other style, and a view with
// one style keeps it and says so.
func TestStyleKey(t *testing.T) {
	var styles []string
	m := newTest(60, 10,
		fakeView{name: "a", table: sample("a"), styles: &styles},
		fakeView{name: "b", table: sample("b"), styles: &styles, style: "current", only: true})
	run(t, m, m.render()) // the first table: later renders use the views
	keyCmd(t, m, "r")
	keyCmd(t, m, "r")
	keyCmd(t, m, "tab")
	keyCmd(t, m, "r")
	if want := []string{"difference", "current", "difference", "current", "current"}; !slices.Equal(styles, want) {
		t.Errorf("styles %v, want %v", styles, want)
	}
	if m.message != "this view has one style: current" {
		t.Errorf("message %q", m.message)
	}
}

// TestAnonymiseKey checks a passes the run's Anonymiser while on, none
// while off.
func TestAnonymiseKey(t *testing.T) {
	var got []presenter.Options
	m := newTest(60, 10, fakeView{name: "a", got: &got})
	m.cfg.Anonymiser = anonymise.New()
	run(t, m, m.render())
	keyCmd(t, m, "a")
	on := m.message
	keyCmd(t, m, "a")
	if len(got) != 3 || got[0].Anonymiser != nil || got[1].Anonymiser != m.cfg.Anonymiser || got[2].Anonymiser != nil {
		t.Errorf("anonymisers %v %v %v", got[0].Anonymiser, got[1].Anonymiser, got[2].Anonymiser)
	}
	if on != "anonymise on" || m.message != "anonymise off" {
		t.Errorf("messages %q, %q", on, m.message)
	}
}
