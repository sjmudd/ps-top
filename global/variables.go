// Package global provides information on global variables and status
package global

import (
	"database/sql"
	"strconv"
	"strings"

	"github.com/sjmudd/ps-top/log"
)

const (
	showCompatibility56ErrorNum    = 3167 // Error 3167: The 'INFORMATION_SCHEMA.GLOBAL_VARIABLES' feature is disabled; see the documentation for 'show_compatibility_56'
	globalVariablesNotInISErrorNum = 1109 // Error 1109: Unknown table 'GLOBAL_VARIABLES' in information_schema

	informationSchemaGlobalVariables = "INFORMATION_SCHEMA.GLOBAL_VARIABLES"
	performanceSchemaGlobalVariables = "performance_schema.global_variables"
)

// We expect to use I_S to query Global Variables. 5.7+ now wants us to use P_S,
// so this variable will be changed if we see the show_compatibility_56 error message

// globally used by Status and Variables
var seenCompatibilityError bool

// may be modified by usePerformanceSchema()
var globalVariablesTable = informationSchemaGlobalVariables // default

// Variables holds the handle and variables collected from the database
type Variables struct {
	db        *sql.DB
	variables map[string]string
}

// shared by Status and Variables
// - no locking. Not sure if absolutely necessary.
func usePerformanceSchema() {
	seenCompatibilityError = true
	globalStatusTable = performanceSchemaGlobalStatus
	globalVariablesTable = performanceSchemaGlobalVariables
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

// NewVariables returns a pointer to an initialised Variables structure with one collection done.
func NewVariables(db *sql.DB) *Variables {
	if db == nil {
		log.Fatal("NewVariables(): db == nil")
	}

	v := &Variables{
		db: db,
	}
	return v.selectAll()
}

// Get returns the value of the given variable if found or an empty string if not.
func (v Variables) Get(key string) string {
	var result string
	var ok bool

	if result, ok = v.variables[key]; !ok {
		result = ""
	}

	return result
}

// selectAll collects all variables from the database and stores for later use.
// - all returned keys are lower-cased.
func (v *Variables) selectAll() *Variables {
	hashref := make(map[string]string)

	query := "SELECT VARIABLE_NAME, VARIABLE_VALUE FROM " + globalVariablesTable
	log.Println("query:", query)

	rows, err := v.db.Query(query)
	if err != nil {
		if !seenCompatibilityError && (IsMysqlError(err, showCompatibility56ErrorNum) || IsMysqlError(err, globalVariablesNotInISErrorNum)) {
			log.Println("selectAll() I_S query failed, trying with P_S")
			usePerformanceSchema()
			query = "SELECT VARIABLE_NAME, VARIABLE_VALUE FROM " + globalVariablesTable
			log.Println("query:", query)

			rows, err = v.db.Query(query)
		}
		if err != nil {
			log.Fatal("selectAll() query", query, "failed with:", err)
		}
	}
	log.Println("selectAll() query succeeded")

	for rows.Next() {
		var variable, value string
		if err := rows.Scan(&variable, &value); err != nil {
			log.Fatal(err)
		}
		hashref[strings.ToLower(variable)] = value
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	_ = rows.Close()

	log.Println("selectAll() result has", len(hashref), "rows")

	v.variables = hashref

	return v
}
