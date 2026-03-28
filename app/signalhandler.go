package app

import (
	"os"
	"os/signal"
	"syscall"
)

// SignalHandler manages signal handling for the application.
type SignalHandler struct {
	sigChan chan os.Signal
}

// NewSignalHandler creates a new signal handler listening for SIGINT and SIGTERM.
func NewSignalHandler() *SignalHandler {
	sh := &SignalHandler{
		sigChan: make(chan os.Signal, 10),
	}
	signal.Notify(sh.sigChan, syscall.SIGINT, syscall.SIGTERM)
	return sh
}

// Channel returns the signal channel (read-only to caller).
func (sh *SignalHandler) Channel() <-chan os.Signal {
	return sh.sigChan
}
