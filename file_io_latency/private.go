// Package file_io_latency contains the routines for
// managing the file_summary_by_instance table.
package file_io_latency

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"sort"
	"time"

	"github.com/sjmudd/ps-top/key_value_cache"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/rc"
)

/*
CREATE TABLE `file_summary_by_instance` (
  `FILE_NAME` varchar(512) NOT NULL,
  `EVENT_NAME` varchar(128) NOT NULL,				// not collected
  `OBJECT_INSTANCE_BEGIN` bigint(20) unsigned NOT NULL,		// not collected
  `COUNT_STAR` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WAIT` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WAIT` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WAIT` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WAIT` bigint(20) unsigned NOT NULL,
  `COUNT_READ` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_READ` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_READ` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_READ` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_READ` bigint(20) unsigned NOT NULL,
  `SUM_NUMBER_OF_BYTES_READ` bigint(20) NOT NULL,
  `COUNT_WRITE` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `SUM_NUMBER_OF_BYTES_WRITE` bigint(20) NOT NULL,
  `COUNT_MISC` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_MISC` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_MISC` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_MISC` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_MISC` bigint(20) unsigned NOT NULL
) ENGINE=PERFORMANCE_SCHEMA DEFAULT CHARSET=utf8
1 row in set (0.00 sec)
*/

//     foo/../bar --> foo/bar   perl: $new =~ s{[^/]+/\.\./}{/};
//     /./        --> /         perl: $new =~ s{/\./}{};
//     //         --> /         perl: $new =~ s{//}{/};
const (
	reEncoded = `@(\d{4})` // FIXME - add me to catch @0024 --> $ for example
)

var (
	reOneOrTheOther    = regexp.MustCompile(`/(\.)?/`)
	reSlashDotDotSlash = regexp.MustCompile(`[^/]+/\.\./`)
	reTableFile        = regexp.MustCompile(`/([^/]+)/([^/]+)\.(frm|ibd|MYD|MYI|CSM|CSV|par)$`)
	reTempTable        = regexp.MustCompile(`#sql-[0-9_]+`)
	rePartTable        = regexp.MustCompile(`(.+)#P#p(\d+|MAX)`)
	reIbdata           = regexp.MustCompile(`/ibdata\d+$`)
	reIbtmp            = regexp.MustCompile(`/ibtmp\d+$`)
	reRedoLog          = regexp.MustCompile(`/ib_logfile\d+$`)
	reBinlog           = regexp.MustCompile(`/binlog\.(\d{6}|index)$`)
	reDbOpt            = regexp.MustCompile(`/db\.opt$`)
	reSlowlog          = regexp.MustCompile(`/slowlog$`)
	reAutoCnf          = regexp.MustCompile(`/auto\.cnf$`)
	rePidFile          = regexp.MustCompile(`/[^/]+\.pid$`)
	reErrorMsg         = regexp.MustCompile(`/share/[^/]+/errmsg\.sys$`)
	reCharset          = regexp.MustCompile(`/share/charsets/Index\.xml$`)
	reDollar           = regexp.MustCompile(`@0024`) // FIXME - add me to catch @0024 --> $ (specific case)

	cache key_value_cache.KeyValueCache
)

// Row contains a row from file_summary_by_instance
type Row struct {
	FILE_NAME string

	COUNT_STAR  uint64
	COUNT_READ  uint64
	COUNT_WRITE uint64
	COUNT_MISC  uint64

	SUM_TIMER_WAIT  uint64
	SUM_TIMER_READ  uint64
	SUM_TIMER_WRITE uint64
	SUM_TIMER_MISC  uint64

	SUM_NUMBER_OF_BYTES_READ  uint64
	SUM_NUMBER_OF_BYTES_WRITE uint64
}

// Rows represents a slice of Row
type Rows []Row

// Return the name using the FILE_NAME attribute.
func (row Row) name() string {
	return row.FILE_NAME
}

func (row Row) headings() string {
	return fmt.Sprintf("%10s %6s|%6s %6s %6s|%8s %8s|%8s %6s %6s %6s|%s",
		"Latency",
		"%",
		"Read",
		"Write",
		"Misc",
		"Rd bytes",
		"Wr bytes",
		"Ops",
		"R Ops",
		"W Ops",
		"M Ops",
		"Table Name")
}

// generate a printable result
func (row Row) rowContent(totals Row) string {
	var name = row.name()

	// We assume that if COUNT_STAR = 0 then there's no data at all...
	// when we have no data we really don't want to show the name either.
	if row.COUNT_STAR == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s|%6s %6s %6s|%8s %8s|%8s %6s %6s %6s|%s",
		lib.FormatTime(row.SUM_TIMER_WAIT),
		lib.FormatPct(lib.MyDivide(row.SUM_TIMER_WAIT, totals.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(row.SUM_TIMER_READ, row.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(row.SUM_TIMER_WRITE, row.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(row.SUM_TIMER_MISC, row.SUM_TIMER_WAIT)),
		lib.FormatAmount(row.SUM_NUMBER_OF_BYTES_READ),
		lib.FormatAmount(row.SUM_NUMBER_OF_BYTES_WRITE),
		lib.FormatAmount(row.COUNT_STAR),
		lib.FormatPct(lib.MyDivide(row.COUNT_READ, row.COUNT_STAR)),
		lib.FormatPct(lib.MyDivide(row.COUNT_WRITE, row.COUNT_STAR)),
		lib.FormatPct(lib.MyDivide(row.COUNT_MISC, row.COUNT_STAR)),
		name)
}

func (row *Row) add(other Row) {
	row.COUNT_STAR += other.COUNT_STAR
	row.COUNT_READ += other.COUNT_READ
	row.COUNT_WRITE += other.COUNT_WRITE
	row.COUNT_MISC += other.COUNT_MISC

	row.SUM_TIMER_WAIT += other.SUM_TIMER_WAIT
	row.SUM_TIMER_READ += other.SUM_TIMER_READ
	row.SUM_TIMER_WRITE += other.SUM_TIMER_WRITE
	row.SUM_TIMER_MISC += other.SUM_TIMER_MISC

	row.SUM_NUMBER_OF_BYTES_READ += other.SUM_NUMBER_OF_BYTES_READ
	row.SUM_NUMBER_OF_BYTES_WRITE += other.SUM_NUMBER_OF_BYTES_WRITE
}

func (row *Row) subtract(other Row) {
	row.COUNT_STAR -= other.COUNT_STAR
	row.COUNT_READ -= other.COUNT_READ
	row.COUNT_WRITE -= other.COUNT_WRITE
	row.COUNT_MISC -= other.COUNT_MISC

	row.SUM_TIMER_WAIT -= other.SUM_TIMER_WAIT
	row.SUM_TIMER_READ -= other.SUM_TIMER_READ
	row.SUM_TIMER_WRITE -= other.SUM_TIMER_WRITE
	row.SUM_TIMER_MISC -= other.SUM_TIMER_MISC

	row.SUM_NUMBER_OF_BYTES_READ -= other.SUM_NUMBER_OF_BYTES_READ
	row.SUM_NUMBER_OF_BYTES_WRITE -= other.SUM_NUMBER_OF_BYTES_WRITE
}

// return the totals of a slice of rows
func (rows Rows) totals() Row {
	var totals Row
	totals.FILE_NAME = "Totals"

	for i := range rows {
		totals.add(rows[i])
	}

	return totals
}

// clean up the given path reducing redundant stuff and return the clean path
func cleanupPath(path string) string {
	for {
		origPath := path
		path = reOneOrTheOther.ReplaceAllString(path, "/")
		path = reSlashDotDotSlash.ReplaceAllString(path, "/")
		if origPath == path { // no change so give up
			break
		}
	}

	return path
}

// From the original FILE_NAME we want to generate a simpler name to use.
// This simpler name may also merge several different filenames into one.
func (row Row) simplifyName(globalVariables map[string]string) string {

	path := row.FILE_NAME

	if cachedResult, err := cache.Get(path); err == nil {
		return cachedResult
	}

	// @0024 --> $ (should do this more generically)
	path = reDollar.ReplaceAllLiteralString(path, "$")

	// this should probably be ordered from most expected regexp to least
	if m1 := reTableFile.FindStringSubmatch(path); m1 != nil {
		// we may match temporary tables so check for them
		if m2 := reTempTable.FindStringSubmatch(m1[2]); m2 != nil {
			return cache.Put(path, "<temp_table>")
		}

		// we may match partitioned tables so check for them
		if m3 := rePartTable.FindStringSubmatch(m1[2]); m3 != nil {
			return cache.Put(path, m1[1]+"."+m3[1]) // <schema>.<table> (less partition info)
		}

		return cache.Put(path, rc.Munge(m1[1]+"."+m1[2])) // <schema>.<table>
	}
	if reIbtmp.MatchString(path) {
		return cache.Put(path, "<ibtmp>")
	}
	if reIbdata.MatchString(path) {
		return cache.Put(path, "<ibdata>")
	}
	if reRedoLog.MatchString(path) {
		return cache.Put(path, "<redo_log>")
	}
	if reBinlog.MatchString(path) {
		return cache.Put(path, "<binlog>")
	}
	if reDbOpt.MatchString(path) {
		return cache.Put(path, "<db_opt>")
	}
	if reSlowlog.MatchString(path) {
		return cache.Put(path, "<slow_log>")
	}
	if reAutoCnf.MatchString(path) {
		return cache.Put(path, "<auto_cnf>")
	}
	// relay logs are a bit complicated. If a full path then easy to
	// identify, but if a relative path we may need to add $datadir,
	// but also if as I do we have a ../blah/somewhere/path then we
	// need to make it match too.
	if len(globalVariables["relay_log"]) > 0 {
		relayLog := globalVariables["relay_log"]
		if relayLog[0] != '/' { // relative path
			relayLog = cleanupPath(globalVariables["datadir"] + relayLog) // datadir always ends in /
		}
		reRelayLog := relayLog + `\.(\d{6}|index)$`
		if regexp.MustCompile(reRelayLog).MatchString(path) {
			return cache.Put(path, "<relay_log>")
		}
	}
	if rePidFile.MatchString(path) {
		return cache.Put(path, "<pid_file>")
	}
	if reErrorMsg.MatchString(path) {
		return cache.Put(path, "<errmsg>")
	}
	if reCharset.MatchString(path) {
		return cache.Put(path, "<charset>")
	}
	// clean up datadir to <datadir>
	if len(globalVariables["datadir"]) > 0 {
		reDatadir := regexp.MustCompile("^" + globalVariables["datadir"])
		path = reDatadir.ReplaceAllLiteralString(path, "<datadir>/")
	}

	return cache.Put(path, path)
}

// Convert the imported "table" to a merged one with merged data.
// Combine all entries with the same "FILE_NAME" by adding their values.
func mergeByTableName(orig Rows, globalVariables map[string]string) Rows {
	start := time.Now()
	t := make(Rows, 0, len(orig))

	m := make(map[string]Row)

	// iterate over source table
	for i := range orig {
		var filename string
		var newRow Row
		origRow := orig[i]

		if origRow.COUNT_STAR > 0 {
			filename = origRow.simplifyName(globalVariables)

			// check if we have an entry in the map
			if _, found := m[filename]; found {
				newRow = m[filename]
			} else {
				newRow.FILE_NAME = filename
			}
			newRow.add(origRow)
			m[filename] = newRow // update the map with the new value
		}
	}

	// add the map contents back into the table
	for _, row := range m {
		t = append(t, row)
	}

	lib.Logger.Println("mergeByTableName() took:", time.Duration(time.Since(start)).String())
	return t
}

// Select the raw data from the database into Rows
// - filter out empty values
// - merge rows with the same name into a single row
// - change FILE_NAME into a more descriptive value.
func selectRows(dbh *sql.DB) Rows {
	var t Rows
	start := time.Now()

	sql := "SELECT FILE_NAME, COUNT_STAR, SUM_TIMER_WAIT, COUNT_READ, SUM_TIMER_READ, SUM_NUMBER_OF_BYTES_READ, COUNT_WRITE, SUM_TIMER_WRITE, SUM_NUMBER_OF_BYTES_WRITE, COUNT_MISC, SUM_TIMER_MISC FROM file_summary_by_instance"

	rows, err := dbh.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var r Row

		if err := rows.Scan(&r.FILE_NAME, &r.COUNT_STAR, &r.SUM_TIMER_WAIT, &r.COUNT_READ, &r.SUM_TIMER_READ, &r.SUM_NUMBER_OF_BYTES_READ, &r.COUNT_WRITE, &r.SUM_TIMER_WRITE, &r.SUM_NUMBER_OF_BYTES_WRITE, &r.COUNT_MISC, &r.SUM_TIMER_MISC); err != nil {
			log.Fatal(err)
		}
		t = append(t, r)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	lib.Logger.Println("selectRows() took:", time.Duration(time.Since(start)).String())

	return t
}

// remove the initial values from those rows where there's a match
// - if we find a row we can't match ignore it
func (rows *Rows) subtract(initial Rows) {
	iByName := make(map[string]int)

	// iterate over rows by name
	for i := range initial {
		iByName[initial[i].name()] = i
	}

	for i := range *rows {
		if _, ok := iByName[(*rows)[i].name()]; ok {
			initialI := iByName[(*rows)[i].name()]
			(*rows)[i].subtract(initial[initialI])
		}
	}
}

func (rows Rows) Len() int      { return len(rows) }
func (rows Rows) Swap(i, j int) { rows[i], rows[j] = rows[j], rows[i] }
func (rows Rows) Less(i, j int) bool {
	return (rows[i].SUM_TIMER_WAIT > rows[j].SUM_TIMER_WAIT) ||
		((rows[i].SUM_TIMER_WAIT == rows[j].SUM_TIMER_WAIT) && (rows[i].FILE_NAME < rows[j].FILE_NAME))
}

func (rows *Rows) sort() {
	sort.Sort(rows)
}

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
func (rows Rows) needsRefresh(t2 Rows) bool {
	myTotals := rows.totals()
	otherTotals := t2.totals()

	return myTotals.SUM_TIMER_WAIT > otherTotals.SUM_TIMER_WAIT
}
