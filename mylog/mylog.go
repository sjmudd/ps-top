// Setup logging to the standard log package on startup
// - this is fine but any fatal errors will not be loggged
// - so provide a mylog.Fatalf() to explicitly write out the error
//   to stderror so it can be seen.
package mylog

import (
	"io"
	"io/ioutil"
	"log"
	"os"
)

// not protected but should not be needed
var loggingEnabled = true

func setLoggingDestination(flags int, destination io.Writer) {
	log.SetFlags(flags)
	log.SetOutput(destination)
}

// Adjust log package default logging based on enable.
// We turn off logging completely if enable == false and enable
// logging to a file otherwise.
func SetupLogging(enable bool, logfile string) {
	if !enable {
		setLoggingDestination(0, ioutil.Discard)
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

func Fatal(v ...interface{}) {
	log.Print(v...)

	setLoggingDestination(log.Ldate|log.Ltime|log.Lshortfile, os.Stderr)
	log.Fatal(v...)
}

func Fatalf(format string, v ...interface{}) {
	log.Printf(format, v...)

	setLoggingDestination(log.Ldate|log.Ltime|log.Lshortfile, os.Stderr)
	log.Fatalf(format, v...)
}

func Fatalln(v ...interface{}) {
	log.Println(v...)

	setLoggingDestination(log.Ldate|log.Ltime|log.Lshortfile, os.Stderr)
	log.Fatalln(v...)
}
