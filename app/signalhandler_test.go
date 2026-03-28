package app

import (
	"testing"
)

// TestSignalHandler_New verifies that NewSignalHandler returns a non-nil handler.
func TestSignalHandler_New(t *testing.T) {
	sh := NewSignalHandler()
	if sh == nil {
		t.Fatal("NewSignalHandler returned nil")
	}
}

// TestSignalHandler_Channel verifies that Channel() returns a non-nil channel.
func TestSignalHandler_Channel(t *testing.T) {
	sh := NewSignalHandler()
	ch := sh.Channel()
	if ch == nil {
		t.Fatal("Channel() returned nil")
	}
}

// TestSignalHandler_MultipleHandlers verifies that multiple handlers can be created
// without interfering (each has its own channel).
func TestSignalHandler_MultipleHandlers(t *testing.T) {
	sh1 := NewSignalHandler()
	sh2 := NewSignalHandler()
	if sh1 == nil || sh2 == nil {
		t.Fatal("Failed to create handlers")
	}
	ch1 := sh1.Channel()
	ch2 := sh2.Channel()
	if ch1 == ch2 {
		t.Error("Handlers should have separate channels")
	}
}
