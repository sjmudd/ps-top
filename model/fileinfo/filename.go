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
func FileInfo2MySQLNames(config Config, rows []Row) []Row {
	start := time.Now()
	rowsByName := make(map[string]Row)

	for _, row := range rows {
		var newRow Row
		newName := filename.Simplify(row.Name, config, rc.Munge, utils.QualifiedTableName)

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

	log.Printf("FileInfo2MySQLNames(): took: %v to convert raw to merged, MySQLified %v rows", time.Duration(time.Since(start)).String(), len(rows), len(rowsByName))
	return newRows
}
