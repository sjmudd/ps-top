package app

import (
	"testing"
	"time"

	"github.com/sjmudd/ps-top/event"
	//_ "github.com/sjmudd/ps-top/pstable"
	//_ "github.com/sjmudd/ps-top/view"
	"github.com/sjmudd/ps-top/wait"
)

// fakeTabler implements pstable.Tabler for tests with minimal behavior.
type fakeTabler struct {
	id string
}

func (f *fakeTabler) Collect()                    {}
func (f *fakeTabler) Description() string         { return f.id }
func (f *fakeTabler) EmptyRowContent() string     { return "" }
func (f *fakeTabler) HaveRelativeStats() bool     { return false }
func (f *fakeTabler) Headings() string            { return "" }
func (f *fakeTabler) FirstCollectTime() time.Time { return time.Time{} }
func (f *fakeTabler) LastCollectTime() time.Time  { return time.Time{} }
func (f *fakeTabler) RowContent() []string        { return nil }
func (f *fakeTabler) ResetStatistics()            {}
func (f *fakeTabler) TotalRowContent() string     { return "" }
func (f *fakeTabler) WantRelativeStats() bool     { return false }

// Note: UpdateCurrentTabler depends on view.SetupAndValidate state which
// initialises global view tables; testing it would require initialising a
// fake DB or stubbing view internals. The following tests focus on
// input-event handling which is pure logic and easy to unit-test.

func TestHandleInputEvent_IncreaseDecreasePollTime(t *testing.T) {
	a := &App{}
	a.waiter = wait.NewWaiter()

	// set initial wait interval to 5s
	a.waiter.SetWaitInterval(5 * time.Second)

	// increase
	evInc := event.Event{Type: event.EventIncreasePollTime}
	a.handleInputEvent(evInc)
	if a.waiter.WaitInterval() != 6*time.Second {
		t.Fatalf("after increase: expected 6s, got %v", a.waiter.WaitInterval())
	}

	// decrease: should go back to 5s
	evDec := event.Event{Type: event.EventDecreasePollTime}
	a.handleInputEvent(evDec)
	if a.waiter.WaitInterval() != 5*time.Second {
		t.Fatalf("after decrease: expected 5s, got %v", a.waiter.WaitInterval())
	}
}

func TestHandleInputEvent_DecreaseAtMinimum(t *testing.T) {
	a := &App{}
	a.waiter = wait.NewWaiter()

	// set to minimum (1s)
	a.waiter.SetWaitInterval(1 * time.Second)

	evDec := event.Event{Type: event.EventDecreasePollTime}
	a.handleInputEvent(evDec)

	if a.waiter.WaitInterval() != 1*time.Second {
		t.Fatalf("decrease at minimum should not reduce below 1s, got %v", a.waiter.WaitInterval())
	}
}
