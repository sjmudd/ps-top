// Package lib minimal logger shared by everyone
package lib

import (
	"log"
	"os"
)

// MyLogger provides a pointer to the logger
var Logger *MyLogger

func init() {
	Logger = new(MyLogger)
	Logger.EnableLogging(false)
}

// MyLogger stores the logger information
type MyLogger struct {
	enabled bool
	logger          *log.Logger
}

// EnableLogging allows me to do this or not
func (logger *MyLogger) EnableLogging(enable bool) bool {
	if logger.enabled == enable {
		return enable // as nothing to do
	}

	oldValue := logger.enabled
	logger.enabled = enable

	if enable {
		logfile := MyName() + ".log"

		file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			log.Fatal("Failed to open log file", logfile, ":", err)
		}
		logger.logger = log.New(file, "", log.Ldate|log.Ltime)
	}
	return oldValue
}

// Println calls passed downstream if we have a valid logger setup
func (logger *MyLogger) Println(v ...interface{}) {
	if logger.logger != nil {
		logger.logger.Println(v)
	}
}
