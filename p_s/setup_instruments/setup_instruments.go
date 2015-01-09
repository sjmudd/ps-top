package setup_instruments

import (
	"database/sql"
	"log"

	"github.com/sjmudd/pstop/lib"
)

// Error 1142: UPDATE command denied to user
const UPDATE_FAILED = "Error 1142"

type setup_instruments_row struct {
	NAME    string
	ENABLED string
	TIMED   string
}

type SetupInstruments struct {
	update_succeeded bool
	rows []setup_instruments_row
}

// Change settings to monitor wait/synch/mutex/%
func (si *SetupInstruments) EnableMutexMonitoring(dbh *sql.DB) {
	si.rows = make([]setup_instruments_row, 0, 100)

	// populate the rows which are not set
	sql := "SELECT NAME, ENABLED, TIMED FROM setup_instruments WHERE NAME LIKE 'wait/synch/mutex/%' AND ( enabled <> 'YES' OR timed <> 'YES' )"

	lib.Logger.Println("Collecting p_s.setup_instruments wait/synch/mutex configuration settings")

	rows, err := dbh.Query(sql)
	if err != nil {
		log.Fatal(err)


	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var r setup_instruments_row
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
	lib.Logger.Println("Updating p_s.setup_instruments to allow wait/synch/mutex configuration")

	count = 0
	for i := range si.rows {
		sql := "UPDATE setup_instruments SET enabled = 'YES', TIMED = 'YES' WHERE NAME = '" + si.rows[i].NAME + "'"
		if _, err := dbh.Exec(sql); err == nil {
			si.update_succeeded = true
		} else {
			if err.Error()[0:10] != UPDATE_FAILED {
				log.Fatal(err)
			}
			break

		}
		count++
	}
	lib.Logger.Println(count, "rows changed in p_s.setup_instruments")
}

// restore any changed rows back to their original state
func (si *SetupInstruments) Restore(dbh *sql.DB) {
	// If the previous update didn't work then don't try to restore
	if ! si.update_succeeded {
		lib.Logger.Println("Not restoring p_s.setup_instruments to its original settings as previous UPDATE had failed")
		return
	} else {
		lib.Logger.Println("Restoring p_s.setup_instruments to its original settings")
	}

	// update the rows which need to be set - do multiple updates but I don't care
	count := 0
	for i := range si.rows {
		sql := "UPDATE setup_instruments SET enabled = '" + si.rows[i].ENABLED + "', TIMED = '" + si.rows[i].TIMED + "' WHERE NAME = '" + si.rows[i].NAME + "'"
		if _, err := dbh.Exec(sql); err != nil {
			log.Fatal(err)
		}
		count++
	}
	lib.Logger.Println(count, "rows changed in p_s.setup_instruments")
}
