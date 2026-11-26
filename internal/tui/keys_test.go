package tui

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/sjmudd/ps-top/v2/internal/anonymise"
	"github.com/sjmudd/ps-top/v2/internal/presenter"
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
	press := func(key string, n int, message string) {
		t.Helper()
		keyCmd(t, m, key)
		if m.n != n || m.message != message {
			t.Errorf("%s: n %d, message %q, want %d and %q", key, m.n, m.message, n, message)
		}
	}
	press("w", 5, "window 5s")
	press("w", 5, "window 5s: the shortest")
	var want []int
	for _, s := range []int{10, 15, 30, 45, 60, 90, 120, 150, 180, 210, 240, 270, 300} {
		press("W", s, "window "+secs(time.Duration(s)*time.Second))
		want = append(want, s)
	}
	press("W", 300, "window 300s: the longest")
	if want = append([]int{5}, want...); !slices.Equal(ctl.windows, want) {
		t.Errorf("SetWindow calls %v, want %v", ctl.windows, want)
	}

	// A collect interval of 2 s: 5 s holds 2 collections, 4 s, the shortest.
	m.cfg.CollectInterval, m.n = 2*time.Second, 5
	press("w", 2, "window 4s")
	press("w", 2, "window 4s: the shortest")

	// A collect interval of 3 s: 10 s holds 3 collections, 9 s. Up from 15 s
	// (5) is 30 s (10).
	m.cfg.CollectInterval, m.n = 3*time.Second, 5
	press("w", 3, "window 9s")
	press("W", 5, "window 15s")
	press("W", 10, "window 30s")

	ctl.err = errors.New("collector stopped")
	keyCmd(t, m, "W")
	if m.n != 10 || m.message != "window: collector stopped" {
		t.Errorf("failed change: n %d, message %q", m.n, m.message)
	}

	m.cfg.Control = nil
	keyCmd(t, m, "W")
	if m.message != "the window cannot be changed" {
		t.Errorf("no control: message %q", m.message)
	}
}

// TestSlowCollection checks a slow collection is shown once, and a later one
// only 30s after the last shown.
func TestSlowCollection(t *testing.T) {
	ctl := &fakeControl{}
	m := newTest(60, 10, fakeView{name: "a"})
	m.cfg.Control, m.cfg.CollectInterval = ctl, time.Second
	clock := t0
	m.now = func() time.Time { return clock }
	tick := func(at time.Duration, took time.Duration, slowAt time.Duration, want string) {
		t.Helper()
		clock = t0.Add(at)
		if took > 0 {
			ctl.slowTook, ctl.slowAt = took, t0.Add(slowAt)
		}
		m.message = ""
		m.Update(tickMsg{})
		if m.message != want {
			t.Errorf("at %v: message %q, want %q", at, m.message, want)
		}
	}
	const msg = "slow collection: 1.2s > 1s. Increase --collect-interval"
	tick(0, 0, 0, "")                                              // none yet
	tick(time.Second, 1200*time.Millisecond, time.Second, msg)     // shown
	tick(2*time.Second, 0, 0, "")                                  // the same one: not again
	tick(10*time.Second, 1200*time.Millisecond, 9*time.Second, "") // a new one within 30s: held back
	tick(31*time.Second, 0, 0, msg)                                // 30s later: shown
	tick(32*time.Second, 0, 0, "")
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
