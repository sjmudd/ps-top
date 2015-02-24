// Manage the configuration of setup_instruments.
package setup_instruments

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/sjmudd/pstop/lib"
)

// constants
const sql_select = "SELECT NAME, ENABLED, TIMED FROM setup_instruments WHERE NAME LIKE ? AND 'YES NOT IN (enabled,timed)"

// We only match on the error number
// Error 1142: UPDATE command denied to user 'cacti'@'10.164.132.182' for table 'setup_instruments'
// Error 1290: The MySQL server is running with the --read-only option so it cannot execute this statement
var EXPECTED_UPDATE_ERRORS = []string{
	"Error 1142:",
	"Error 1290:",
}

// one row of performance_schema.setup_instruments
type table_row struct {
	NAME    string
	ENABLED string
	TIMED   string
}

type table_rows []table_row

// SetupInstruments "object"
type SetupInstruments struct {
	update_tried     bool
	update_succeeded bool
	rows             table_rows
	dbh              *sql.DB
}

// Return a newly initialised SetupInstruments structure with a handle to the database.
// Better to return a pointer ?
func NewSetupInstruments(dbh *sql.DB) SetupInstruments {
	return SetupInstruments{dbh: dbh}
}

// enable mutex and stage monitoring
func (si *SetupInstruments) EnableMonitoring() {
	si.EnableMutexMonitoring()
	si.EnableStageMonitoring()
}

// Change settings to monitor stage/sql/%
func (si *SetupInstruments) EnableStageMonitoring() {
	lib.Logger.Println("EnableStageMonitoring")
	sql_match := "stage/sql/%"
	collecting := "Collecting setup_instruments stage/sql configuration settings"
	updating := "Updating setup_instruments configuration for: stage/sql"

	si.Configure(sql_match, collecting, updating)
	lib.Logger.Println("EnableStageMonitoring finishes")
}

// Change settings to monitor wait/synch/mutex/%
func (si *SetupInstruments) EnableMutexMonitoring() {
	lib.Logger.Println("EnableMutexMonitoring")
	sql_match := "wait/synch/mutex/%"
	collecting := "Collecting setup_instruments wait/synch/mutex configuration settings"
	updating := "Updating setup_instruments configuration for: wait/synch/mutex"

	si.Configure(sql_match, collecting, updating)
	lib.Logger.Println("EnableMutexMonitoring finishes")
}

// return true if the error is not in the expected list
func error_in_expected_list(actual_error string, expected_errors []string) bool {
	lib.Logger.Println("checking if", actual_error, "is in", expected_errors)
	e := actual_error[0:11]
	expected_error := false
	for i := range expected_errors {
		if e == expected_errors[i] {
			lib.Logger.Println("found an expected error", expected_errors[i])
			expected_error = true
			break
		}
	}
	lib.Logger.Println("returning", expected_error)
	return expected_error
}

// generic routine (now) to update some rows in setup instruments
func (si *SetupInstruments) Configure(sql_match string, collecting, updating string) {
	lib.Logger.Println(fmt.Sprintf("Configure(%q,%q,%q)", sql_match, collecting, updating))
	// skip if we've tried and failed
	if si.update_tried && !si.update_succeeded {
		lib.Logger.Println("Configure() - Skipping further configuration")
		return
	}

	// setup the old values in case they're not set
	if si.rows == nil {
		si.rows = make([]table_row, 0, 500)
	}

	lib.Logger.Println(collecting)

	lib.Logger.Println("dbh.query", sql_select, sql_match)
	rows, err := si.dbh.Query(sql_select, sql_match)
	if err != nil {
		log.Fatal(err)
	}

	count := 0
	for rows.Next() {
		var r table_row
		if err := rows.Scan(
			&r.NAME,
			&r.ENABLED,
			&r.TIMED); err != nil {
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

	const update_sql = "UPDATE setup_instruments SET enabled = ?, TIMED = ? WHERE NAME = ?"
	lib.Logger.Println("Preparing statement:", update_sql)
	si.update_tried = true
	lib.Logger.Println("dbh.Prepare", update_sql)
	stmt, err := si.dbh.Prepare(update_sql)
	if err != nil {
		lib.Logger.Println("- prepare gave error:", err.Error())
		if !error_in_expected_list(err.Error(), EXPECTED_UPDATE_ERRORS) {
			log.Fatal("Not expected error so giving up")
		} else {
			lib.Logger.Println("- expected error so not running statement")
		}
	} else {
		lib.Logger.Println("Prepare succeeded, trying to update", len(si.rows), "row(s)")
		count = 0
		for i := range si.rows {
			lib.Logger.Println("- changing row:", si.rows[i].NAME)
			lib.Logger.Println("stmt.Exec", "YES", "YES", si.rows[i].NAME)
			if res, err := stmt.Exec("YES", "YES", si.rows[i].NAME); err == nil {
				lib.Logger.Println("update succeeded")
				si.update_succeeded = true
				c, _ := res.RowsAffected()
				count += int(c)
			} else {
				si.update_succeeded = false
				if error_in_expected_list(err.Error(), EXPECTED_UPDATE_ERRORS) {
					lib.Logger.Println("Insufficient privileges to UPDATE setup_instruments: " + err.Error())
					lib.Logger.Println("Not attempting further updates")
					return
				} else {
					log.Fatal(err)
				}
			}
		}
		if si.update_succeeded {
			lib.Logger.Println(count, "rows changed in p_s.setup_instruments")
		}
		stmt.Close()
	}
	lib.Logger.Println("Configure() returns update_tried", si.update_tried, ", update_succeeded", si.update_succeeded)
}

// restore setup_instruments rows to their previous settings
func (si *SetupInstruments) RestoreConfiguration() {
	lib.Logger.Println("RestoreConfiguration()")
	// If the previous update didn't work then don't try to restore
	if !si.update_succeeded {
		lib.Logger.Println("Not restoring p_s.setup_instruments to original settings as initial configuration attempt failed")
		return
	} else {
		lib.Logger.Println("Restoring p_s.setup_instruments to its original settings")
	}

	// update the rows which need to be set - do multiple updates but I don't care
	update_sql := "UPDATE setup_instruments SET enabled = ?, TIMED = ? WHERE NAME = ?"
	lib.Logger.Println("dbh.Prepare(", update_sql, ")")
	stmt, err := si.dbh.Prepare(update_sql)
	if err != nil {
		log.Fatal(err)
	}
	count := 0
	for i := range si.rows {
		lib.Logger.Println("stmt.Exec(", si.rows[i].ENABLED, si.rows[i].TIMED, si.rows[i].NAME, ")")
		if _, err := stmt.Exec(si.rows[i].ENABLED, si.rows[i].TIMED, si.rows[i].NAME); err != nil {
			log.Fatal(err)
		}
		count++
	}
	lib.Logger.Println("stmt.Close()")
	stmt.Close()
	lib.Logger.Println(count, "rows changed in p_s.setup_instruments")
}
