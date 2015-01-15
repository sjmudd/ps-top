// manage the configuration of performance_schema.setup_instruments
package setup_instruments

import (
	"database/sql"
	"log"

	"github.com/sjmudd/pstop/lib"
)

// We only match on the error number
// Error 1142: UPDATE command denied to user
// Error 1290: The MySQL server is running with the --read-only option so it cannot execute this statement
var EXPECTED_UPDATE_ERRORS = []string{
	"Error 1142",
	"Error 1290",
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
	update_succeeded bool
	rows             table_rows
}

// Change settings to monitor stage/sql/%
func (si *SetupInstruments) EnableStageMonitoring(dbh *sql.DB) {
	sql := "SELECT NAME, ENABLED, TIMED FROM setup_instruments WHERE NAME LIKE 'stage/sql/%' AND ( enabled <> 'YES' OR timed <> 'YES' )"
	collecting := "Collecting p_s.setup_instruments stage/sql configuration settings"
	updating := "Updating p_s.setup_instruments to allow stage/sql configuration"

	si.ConfigureSetupInstruments(dbh, sql, collecting, updating)
}

// Change settings to monitor wait/synch/mutex/%
func (si *SetupInstruments) EnableMutexMonitoring(dbh *sql.DB) {
	sql := "SELECT NAME, ENABLED, TIMED FROM setup_instruments WHERE NAME LIKE 'wait/synch/mutex/%' AND ( enabled <> 'YES' OR timed <> 'YES' )"
	collecting := "Collecting p_s.setup_instruments wait/synch/mutex configuration settings"
	updating := "Updating p_s.setup_instruments to allow wait/synch/mutex configuration"

	si.ConfigureSetupInstruments(dbh, sql, collecting, updating)
}

// generic routine (now) to update some rows in setup instruments
func (si *SetupInstruments) ConfigureSetupInstruments(dbh *sql.DB, sql string, collecting, updating string) {
	// setup the old values in case they're not set
	if si.rows == nil {
		si.rows = make([]table_row, 0, 500)
	}

	lib.Logger.Println(collecting)

	rows, err := dbh.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var r table_row
		if err := rows.Scan(
			&r.NAME,
			&r.ENABLED,
			&r.TIMED); err != nil {
			log.Fatal(err)
		}
		// we collect all information even if it's mainly empty as we may reference it later
		si.rows = append(si.rows, r)
		count++
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	lib.Logger.Println("- found", count, "rows whose configuration need changing")

	// update the rows which need to be set - do multiple updates but I don't care
	lib.Logger.Println(updating)

	count = 0
	update_sql := "UPDATE setup_instruments SET enabled = ?, TIMED = ? WHERE NAME = ?"
	stmt, err := dbh.Prepare( update_sql )
	if err != nil {
		log.Fatal(err)
	}
	for i := range si.rows {
		if _, err := stmt.Exec(update_sql, "YES", "YES", si.rows[i].NAME); err == nil {
			si.update_succeeded = true
		} else {
			found_expected := false
			for i := range EXPECTED_UPDATE_ERRORS {
				if err.Error()[0:10] == EXPECTED_UPDATE_ERRORS[i] {
					found_expected = true
					break
				}
			}
			if !found_expected {
				log.Fatal(err)
			}
			lib.Logger.Println("Insufficient privileges to UPDATE setup_instruments: " + err.Error())
			break
		}
		count++
	}
	if si.update_succeeded {
		lib.Logger.Println(count, "rows changed in p_s.setup_instruments")
	}
}

// restore setup_instruments rows to their previous settings
func (si *SetupInstruments) RestoreConfiguration(dbh *sql.DB) {
	// If the previous update didn't work then don't try to restore
	if !si.update_succeeded {
		lib.Logger.Println("Not restoring p_s.setup_instruments to its original settings as previous UPDATE had failed")
		return
	} else {
		lib.Logger.Println("Restoring p_s.setup_instruments to its original settings")
	}

	// update the rows which need to be set - do multiple updates but I don't care
	count := 0
	update_sql := "UPDATE setup_instruments SET enabled = ?, TIMED = ? WHERE NAME = ?"
	stmt, err := dbh.Prepare( update_sql )
	if err != nil {
		log.Fatal(err)
	}
	for i := range si.rows {
		if _, err := stmt.Exec(update_sql, si.rows[i].ENABLED, si.rows[i].TIMED, si.rows[i].NAME ); err != nil {
			log.Fatal(err)
		}
		count++
	}
	lib.Logger.Println(count, "rows changed in p_s.setup_instruments")
}
