// Minimal logger shared by everyone
package logger

import (
	"log"
	"os"
)

var (
	enabled bool
	logger  *log.Logger
	logfile string
)

func Disable() bool {
	oldValue := enabled
	enabled = false

	return oldValue
}

// EnableLogging allows me to do this or not
func Enable(filename string) bool {
	if enabled {
		return enabled // as nothing to do
	}

	oldValue := enabled

	enabled = true
	logfile = filename

	file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		log.Fatal("Failed to open log file", logfile, ":", err)
	}
	logger = log.New(file, "", log.Ldate|log.Ltime)

	return oldValue
}

// Println calls passed downstream if we have a valid logger setup
func Println(v ...interface{}) {
	if logger != nil {
		logger.Println(v)
	}
}
