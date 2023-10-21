// Package mylog provides some adjustments to the standard logging
// = it is called on startup to optionally stop all logging.
// - use mylog.Fatal*(...) to ensure that this is always logged.
package mylog

import (
	"io"
	"log"
	"os"
)

func setLoggingDestination(flags int, destination io.Writer) {
	log.SetFlags(flags)
	log.SetOutput(destination)
}

// SetupLogging adjusts the log package default logging based on enable.
// We turn off logging completely if enable == false and enable
// logging to a file otherwise.
func SetupLogging(enable bool, logfile string) {
	if !enable {
		setLoggingDestination(0, io.Discard)
		return
	}

	file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
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
func Fatal(v ...interface{}) {
	log.Print(v...)

	setLoggingDestination(log.Ldate|log.Ltime|log.Lshortfile, os.Stderr)
	log.Fatal(v...)
}

// Fatalf logs to file (if enabled) and also to stderr
func Fatalf(format string, v ...interface{}) {
	log.Printf(format, v...)

	setLoggingDestination(log.Ldate|log.Ltime|log.Lshortfile, os.Stderr)
	log.Fatalf(format, v...)
}

// Fatalln logs to file (if enabled) and also to stderr
func Fatalln(v ...interface{}) {
	log.Println(v...)

	setLoggingDestination(log.Ldate|log.Ltime|log.Lshortfile, os.Stderr)
	log.Fatalln(v...)
}
