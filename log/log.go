// Package log provides some adjustments to the standard logging
// = it is called on startup to optionally stop all logging.
// - use log.Fatal*(...) to ensure that this is always logged.
package log

import (
	"io"
	"log"
	"os"
)

func setLoggingDestination(flags int, destination io.Writer) {
	log.SetFlags(flags)
	log.SetOutput(destination)
}

func setOutputOnly(destination io.Writer) {
	log.SetOutput(destination)
}

// loggingEnabled tracks whether file/stderr logging was enabled via SetupLogging.
var loggingEnabled bool

// SetupLogging adjusts the log package default logging based on enable.
// We turn off logging completely if enable == false and enable
// logging to a file otherwise.
func SetupLogging(enable bool, logfile string) {
	loggingEnabled = enable
	if !enable {
		setLoggingDestination(0, io.Discard)
		return
	}

	file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		// If we can't open the file, write the error to stderr so the user sees it.
		// Use the stdlib log directly as logging hasn't been configured yet.
		log.SetOutput(os.Stderr)
		log.Fatalf("Failed to open log file %q: %v", logfile, err)
	}

	setLoggingDestination(log.Ldate|log.Ltime|log.Lshortfile, file)
}

// if logging is enabled it is sent to to a file which will not be visible.
// If logging is disabled nothing will be logged.
// Neither option is good for the user as he/she will see nothing.
// So write loggging as configured and then write to stderr where the
// user will see it.

// Fatal logs to file (if enabled) and also to stderr
func Fatal(v ...any) {
	// Always attempt to write the message using the configured logger (may be discarded).
	log.Print(v...)

	// If logging was disabled, ensure the user sees the fatal message on stderr
	// and then exit. If logging is enabled the MultiWriter will already include stderr.
	if !loggingEnabled {
		setOutputOnly(os.Stderr)
	}
	log.Fatal(v...)
}

// Fatalf logs to file (if enabled) and also to stderr
func Fatalf(format string, v ...any) {
	log.Printf(format, v...)
	if !loggingEnabled {
		setOutputOnly(os.Stderr)
	}
	log.Fatalf(format, v...)
}

// Fatalln logs to file (if enabled) and also to stderr
func Fatalln(v ...any) {
	log.Println(v...)
	if !loggingEnabled {
		setOutputOnly(os.Stderr)
	}
	log.Fatalln(v...)
}

// Println provides the same interface as log.Println
func Println(v ...any) {
	log.Println(v...)
}

// Printf provides the same interface as log.Printf
func Printf(format string, v ...any) {
	log.Printf(format, v...)
}
