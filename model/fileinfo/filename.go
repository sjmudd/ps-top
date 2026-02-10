package fileinfo

import (
	"log"
	"time"

	"github.com/sjmudd/ps-top/filename"
	"github.com/sjmudd/ps-top/rc"
	"github.com/sjmudd/ps-top/utils"
)

// FileInfo2MySQLNames converts the raw imported rows by converting
// filenames to MySQL Object Names, merging similar names together,
// returning the resultant rows.
func FileInfo2MySQLNames(datadir string, relaylog string, rows []Row) []Row {
	start := time.Now()
	rowsByName := make(map[string]Row)

	for _, row := range rows {
		var newRow Row
		newName := filename.Simplify(row.Name, rc.Munge, utils.QualifiedTableName, datadir, relaylog)

		// check if we have an entry in the map
		if _, found := rowsByName[newName]; found {
			newRow = rowsByName[newName]
		} else {
			newRow = Row{Name: newName} // empty row with new name
		}
		newRow = add(newRow, row)
		rowsByName[newName] = newRow // update the map with the new summed row
	}

	// create rows based on the current merged map
	var newRows []Row
	for _, row := range rowsByName {
		newRows = append(newRows, row)
	}

	log.Printf("FileInfo2MySQLNames(): took: %v to convert %v raw rows to merged, MySQLified %v rows",
		time.Since(start),
		len(rows),
		len(rowsByName),
	)
	return newRows
}
