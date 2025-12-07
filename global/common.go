// Package global provides information on global variables and status
package global

import (
	"strconv"
	"strings"
)

const (
	showCompatibility56ErrorNum = 3167 // Error 3167: The 'INFORMATION_SCHEMA.GLOBAL_VARIABLES' feature is disabled; see the documentation for 'show_compatibility_56'
	variablesNotInISErrorNum    = 1109 // Error 1109: Unknown table 'GLOBAL_VARIABLES' in information_schema
)

// We expect to use I_S to query Global Variables. 5.7+ now wants us to use P_S,
// so this variable will be changed if we see the show_compatibility_56 error message

// globally used by Status and Variables
var seenCompatibilityError bool

// shared by Status and Variables
// - no locking. Not sure if absolutely necessary.
func usePerformanceSchema() {
	seenCompatibilityError = true
	statusTable = performanceSchemaGlobalStatus
	variablesTable = performanceSchemaGlobalVariables
}

// IsMysqlError returns true if the given error matches the expected number
//   - format of MySQL error messages changed in database-sql-driver/mysql v1.7.0
//     so adjusting code to handle the expected format
//
// Error 1109 (42S02): Unknown table 'GLOBAL_VARIABLES' in information_schema
func IsMysqlError(err error, wantedErrNum int) bool {
	s := err.Error()
	if len(s) < 19 {
		return false
	}
	if !strings.HasPrefix(s, "Error ") {
		return false
	}
	if s[10] != ' ' {
		return false
	}
	if s[18] != ':' {
		return false
	}
	errNumStr := s[6:10]
	num, err := strconv.Atoi(errNumStr)
	if err != nil {
		return false
	}
	return num == wantedErrNum
}
