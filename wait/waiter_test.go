package wait

import (
	"testing"
	"time"
)

// mockTime creates a timeNow function that returns a fixed time
func mockTime(t time.Time) timeNow {
	return func() time.Time {
		return t
	}
}

func TestWaitInterval(t *testing.T) {
	h := NewWaiter()
	h.SetWaitInterval(5 * time.Second)
	if h.WaitInterval() != 5*time.Second {
		t.Errorf("Expected wait interval of 5s, got %v", h.WaitInterval())
	}
}

func TestSetWaitInterval(t *testing.T) {
	h := NewWaiter()
	h.SetWaitInterval(3 * time.Second)
	if h.collectInterval != 3*time.Second {
		t.Errorf("Expected collection interval of 3s, got %v", h.collectInterval)
	}
}

func TestSetAndLastCollected(t *testing.T) {
	fixedTime := time.Date(2025, 10, 29, 10, 0, 0, 0, time.UTC)
	h := NewWaiter()
	h.SetCollected(fixedTime)
	if !h.LastCollected().Equal(fixedTime) {
		t.Errorf("Expected last collected time %v, got %v", fixedTime, h.LastCollected())
	}
}

func TestCollectedNow(t *testing.T) {
	fixedTime := time.Date(2025, 10, 29, 10, 0, 0, 0, time.UTC)
	h := NewWaiter()
	h.now = mockTime(fixedTime)
	h.CollectedNow()

	if !h.LastCollected().Equal(fixedTime) {
		t.Errorf("Expected last collected time %v, got %v", fixedTime, h.LastCollected())
	}
}

func TestTimeToWait(t *testing.T) {
	baseTime := time.Date(2025, 10, 29, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		lastCollected time.Time
		currentTime   time.Time
		interval      time.Duration
		expectedWait  time.Duration
	}{
		{
			name:          "future collection time",
			lastCollected: baseTime,
			currentTime:   baseTime,
			interval:      5 * time.Second,
			expectedWait:  5 * time.Second,
		},
		{
			name:          "past collection time",
			lastCollected: baseTime,
			currentTime:   baseTime.Add(10 * time.Second),
			interval:      5 * time.Second,
			expectedWait:  extraDelay,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewWaiter()
			h.now = mockTime(tt.currentTime)
			h.lastCollected = tt.lastCollected
			h.collectInterval = tt.interval

			wait := h.TimeToWait()
			if wait != tt.expectedWait {
				t.Errorf("TimeToWait() = %v, want %v", wait, tt.expectedWait)
			}
		})
	}
}

func TestWaitUntilNextPeriod(t *testing.T) {
	baseTime := time.Date(2025, 10, 29, 10, 0, 0, 0, time.UTC)
	h := NewWaiter()
	h.now = mockTime(baseTime)
	h.lastCollected = baseTime.Add(-10 * time.Second)
	h.collectInterval = 1 * time.Second

	ch := h.WaitUntilNextPeriod()
	select {
	case <-ch:
		// Channel received a value as expected
	case <-time.After(300 * time.Millisecond): // slightly more than extraDelay
		t.Error("WaitUntilNextPeriod did not return within expected time")
	}
}
