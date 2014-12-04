// logger - minimal logger shared by everyone
package lib

import (
	"log"
	"os"
)

// public visible interface
var Logger *MyLogger

func init() {
	Logger = new(MyLogger)
	Logger.EnableLogging(false)
}

// just add an extra field to enable or not
type MyLogger struct {
	logging_enabled bool
	logger          *log.Logger
}

// Enable logging to the log file
func (logger *MyLogger) EnableLogging(enable_logging bool) bool {
	if logger.logging_enabled == enable_logging {
		return enable_logging // as nothing to do
	}

	old_value := logger.logging_enabled
	logger.logging_enabled = enable_logging

	if enable_logging {
		logfile := MyName() + ".log"

		file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			log.Fatal("Failed to open log file", logfile, ":", err)
		}
		logger.logger = log.New(file, "", log.Ldate|log.Ltime)
	}
	return old_value
}

// pass Println() calls downstream if we have a valid logger setup
func (logger *MyLogger) Println(v ...interface{}) {
	if logger.logger != nil {
		logger.logger.Println(v)
	}
}
