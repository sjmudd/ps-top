// Package setup_instruments Manages the configuration of performance_schema.setup_instruments.
package setup_instruments

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/sjmudd/ps-top/lib"
)

// constants
const sqlSelect = "SELECT NAME, ENABLED, TIMED FROM setup_instruments WHERE NAME LIKE ? AND 'YES NOT IN (ENABLED,TIMED)"

// We only match on the error number
// Error 1142: UPDATE command denied to user 'myuser'@'10.11.12.13' for table 'setup_instruments'
// Error 1290: The MySQL server is running with the --read-only option so it cannot execute this statement
var ExpectedUpdateErrors = []string{
	"Error 1142:",
	"Error 1290:",
}

// Row contains one row of performance_schema.setup_instruments
type Row struct {
	name    string
	enabled string
	timed   string
}

// Rows contains a slice of Row
type Rows []Row

// SetupInstruments "object"
type SetupInstruments struct {
	updateTried     bool
	updateSucceeded bool
	rows            Rows
	dbh             *sql.DB
}

// NewSetupInstruments returns a newly initialised SetupInstruments
// structure with a handle to the database.  Better to return a
// pointer ?
func NewSetupInstruments(dbh *sql.DB) SetupInstruments {
	return SetupInstruments{dbh: dbh}
}

// EnableMonitoring enables mutex and stage monitoring
func (si *SetupInstruments) EnableMonitoring() {
	si.EnableMutexMonitoring()
	si.EnableStageMonitoring()
}

// EnableStageMonitoring change settings to monitor stage/sql/%
func (si *SetupInstruments) EnableStageMonitoring() {
	lib.Logger.Println("EnableStageMonitoring")
	sqlMatch := "stage/sql/%"
	sqlSelect := "SELECT NAME, ENABLED, TIMED FROM setup_instruments WHERE NAME LIKE '" + sqlMatch + "' AND 'YES' NOT IN (ENABLED,TIMED)"

	collecting := "Collecting setup_instruments stage/sql configuration settings"
	updating := "Updating setup_instruments configuration for: stage/sql"

	si.Configure(sqlSelect, collecting, updating)
	lib.Logger.Println("EnableStageMonitoring finishes")
}

// EnableMutexMonitoring changes settings to monitor wait/synch/mutex/%
func (si *SetupInstruments) EnableMutexMonitoring() {
	lib.Logger.Println("EnableMutexMonitoring")
	sqlMatch := "wait/synch/mutex/%"
	sqlSelect := "SELECT NAME, ENABLED, TIMED FROM setup_instruments WHERE NAME LIKE '" + sqlMatch + "' AND 'YES' NOT IN (ENABLED,TIMED)"
	collecting := "Collecting setup_instruments wait/synch/mutex configuration settings"
	updating := "Updating setup_instruments configuration for: wait/synch/mutex"

	si.Configure(sqlSelect, collecting, updating)
	lib.Logger.Println("EnableMutexMonitoring finishes")
}

// return true if the error is not in the expected list
func errorInExpectedList(actualError string, expectedErrors []string) bool {
	lib.Logger.Println("checking if", actualError, "is in", expectedErrors)
	e := actualError[0:11]
	expectedError := false
	for i := range expectedErrors {
		if e == expectedErrors[i] {
			lib.Logger.Println("found an expected error", expectedErrors[i])
			expectedError = true
			break
		}
	}
	lib.Logger.Println("returning", expectedError)
	return expectedError
}

// Configure updates setup_instruments so we can monitor tables correctly.
func (si *SetupInstruments) Configure(sqlSelect string, collecting, updating string) {
	const updateSQL = "UPDATE setup_instruments SET enabled = ?, TIMED = ? WHERE NAME = ?"

	lib.Logger.Println(fmt.Sprintf("Configure(%q,%q,%q)", sqlSelect, collecting, updating))
	// skip if we've tried and failed
	if si.updateTried && !si.updateSucceeded {
		lib.Logger.Println("Configure() - Skipping further configuration")
		return
	}

	// setup the old values in case they're not set
	if si.rows == nil {
		si.rows = make([]Row, 0, 500)
	}

	lib.Logger.Println(collecting)

	lib.Logger.Println("dbh.query", sqlSelect)
	rows, err := si.dbh.Query(sqlSelect)
	if err != nil {
		log.Fatal(err)
	}

	count := 0
	for rows.Next() {
		var r Row
		if err := rows.Scan(
			&r.name,
			&r.enabled,
			&r.timed); err != nil {
			log.Fatal(err)
		}
		si.rows = append(si.rows, r)
		count++
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	rows.Close()
	lib.Logger.Println("- found", count, "rows whose configuration need changing")

	// update the rows which need to be set - do multiple updates but I don't care
	lib.Logger.Println(updating)

	lib.Logger.Println("Preparing statement:", updateSQL)
	si.updateTried = true
	lib.Logger.Println("dbh.Prepare", updateSQL)
	stmt, err := si.dbh.Prepare(updateSQL)
	if err != nil {
		lib.Logger.Println("- prepare gave error:", err.Error())
		if !errorInExpectedList(err.Error(), ExpectedUpdateErrors) {
			log.Fatal("Not expected error so giving up")
		} else {
			lib.Logger.Println("- expected error so not running statement")
		}
	} else {
		lib.Logger.Println("Prepare succeeded, trying to update", len(si.rows), "row(s)")
		count = 0
		for i := range si.rows {
			lib.Logger.Println("- changing row:", si.rows[i].name)
			lib.Logger.Println("stmt.Exec", "YES", "YES", si.rows[i].name)
			if res, err := stmt.Exec("YES", "YES", si.rows[i].name); err == nil {
				lib.Logger.Println("update succeeded")
				si.updateSucceeded = true
				c, _ := res.RowsAffected()
				count += int(c)
			} else {
				si.updateSucceeded = false
				if errorInExpectedList(err.Error(), ExpectedUpdateErrors) {
					lib.Logger.Println("Insufficient privileges to UPDATE setup_instruments: " + err.Error())
					lib.Logger.Println("Not attempting further updates")
					return
				}
				log.Fatal(err)
			}
		}
		if si.updateSucceeded {
			lib.Logger.Println(count, "rows changed in p_s.setup_instruments")
		}
		stmt.Close()
	}
	lib.Logger.Println("Configure() returns updateTried", si.updateTried, ", updateSucceeded", si.updateSucceeded)
}

// RestoreConfiguration restores setup_instruments rows to their previous settings (if changed previously).
func (si *SetupInstruments) RestoreConfiguration() {
	lib.Logger.Println("RestoreConfiguration()")
	// If the previous update didn't work then don't try to restore
	if !si.updateSucceeded {
		lib.Logger.Println("Not restoring p_s.setup_instruments to original settings as initial configuration attempt failed")
		return
	}
	lib.Logger.Println("Restoring p_s.setup_instruments to its original settings")

	// update the rows which need to be set - do multiple updates but I don't care
	updateSQL := "UPDATE setup_instruments SET enabled = ?, TIMED = ? WHERE NAME = ?"
	lib.Logger.Println("dbh.Prepare(", updateSQL, ")")
	stmt, err := si.dbh.Prepare(updateSQL)
	if err != nil {
		log.Fatal(err)
	}
	count := 0
	for i := range si.rows {
		lib.Logger.Println("stmt.Exec(", si.rows[i].enabled, si.rows[i].timed, si.rows[i].name, ")")
		if _, err := stmt.Exec(si.rows[i].enabled, si.rows[i].timed, si.rows[i].name); err != nil {
			log.Fatal(err)
		}
		count++
	}
	lib.Logger.Println("stmt.Close()")
	stmt.Close()
	lib.Logger.Println(count, "rows changed in p_s.setup_instruments")
}
