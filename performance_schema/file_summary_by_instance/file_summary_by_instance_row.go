// This file contains the library routines for managing the
// file_summary_by_instance table.
package file_summary_by_instance

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"sort"
	"time"

	"github.com/sjmudd/pstop/key_value_cache"
	"github.com/sjmudd/pstop/lib"
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
	re_encoded = `@(\d{4})` // FIXME - add me to catch @0024 --> $ for example
)

var (
	re_one_or_the_other    *regexp.Regexp = regexp.MustCompile(`/(\.)?/`)
	re_slash_dot_dot_slash *regexp.Regexp = regexp.MustCompile(`[^/]+/\.\./`)
	re_table_file          *regexp.Regexp = regexp.MustCompile(`/([^/]+)/([^/]+)\.(frm|ibd|MYD|MYI|CSM|CSV|par)$`)
	re_temp_table          *regexp.Regexp = regexp.MustCompile(`#sql-[0-9_]+`)
	re_part_table          *regexp.Regexp = regexp.MustCompile(`(.+)#P#p(\d+|MAX)`)
	re_ibdata              *regexp.Regexp = regexp.MustCompile(`/ibdata\d+$`)
	re_redo_log            *regexp.Regexp = regexp.MustCompile(`/ib_logfile\d+$`)
	re_binlog              *regexp.Regexp = regexp.MustCompile(`/binlog\.(\d{6}|index)$`)
	re_db_opt              *regexp.Regexp = regexp.MustCompile(`/db\.opt$`)
	re_slowlog             *regexp.Regexp = regexp.MustCompile(`/slowlog$`)
	re_auto_cnf            *regexp.Regexp = regexp.MustCompile(`/auto\.cnf$`)
	re_pid_file            *regexp.Regexp = regexp.MustCompile(`/[^/]+\.pid$`)
	re_error_msg           *regexp.Regexp = regexp.MustCompile(`/share/[^/]+/errmsg\.sys$`)
	re_charset             *regexp.Regexp = regexp.MustCompile(`/share/charsets/Index\.xml$`)
	re_dollar              *regexp.Regexp = regexp.MustCompile(`@0024`) // FIXME - add me to catch @0024 --> $ (specific case)

	cache key_value_cache.KeyValueCache
)

type file_summary_by_instance_row struct {
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

// represents a table or set of rows
type file_summary_by_instance_rows []file_summary_by_instance_row

// Return the name using the FILE_NAME attribute.
func (r *file_summary_by_instance_row) name() string {
	return r.FILE_NAME
}

// Return a formatted pretty name for the row.
func (r *file_summary_by_instance_row) pretty_name() string {
	s := r.name()
	if len(s) > 30 {
		s = s[:29]
	}
	return fmt.Sprintf("%-30s", s)
}

func (r *file_summary_by_instance_row) headings() string {
	return fmt.Sprintf("%-30s %10s %6s|%6s %6s %6s|%8s %8s|%8s %6s %6s %6s",
		"Table Name",
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
		"M Ops")
}

// generate a printable result
func (row *file_summary_by_instance_row) row_content(totals file_summary_by_instance_row) string {
	var name string

	// We assume that if COUNT_STAR = 0 then there's no data at all...
	// when we have no data we really don't want to show the name either.
	if row.COUNT_STAR == 0 {
		name = ""
	} else {
		name = row.pretty_name()
	}

	return fmt.Sprintf("%-30s %10s %6s|%6s %6s %6s|%8s %8s|%8s %6s %6s %6s",
		name,
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
		lib.FormatPct(lib.MyDivide(row.COUNT_MISC, row.COUNT_STAR)))
}

func (this *file_summary_by_instance_row) add(other file_summary_by_instance_row) {
	this.COUNT_STAR += other.COUNT_STAR
	this.COUNT_READ += other.COUNT_READ
	this.COUNT_WRITE += other.COUNT_WRITE
	this.COUNT_MISC += other.COUNT_MISC

	this.SUM_TIMER_WAIT += other.SUM_TIMER_WAIT
	this.SUM_TIMER_READ += other.SUM_TIMER_READ
	this.SUM_TIMER_WRITE += other.SUM_TIMER_WRITE
	this.SUM_TIMER_MISC += other.SUM_TIMER_MISC

	this.SUM_NUMBER_OF_BYTES_READ += other.SUM_NUMBER_OF_BYTES_READ
	this.SUM_NUMBER_OF_BYTES_WRITE += other.SUM_NUMBER_OF_BYTES_WRITE
}

func (this *file_summary_by_instance_row) subtract(other file_summary_by_instance_row) {
	this.COUNT_STAR -= other.COUNT_STAR
	this.COUNT_READ -= other.COUNT_READ
	this.COUNT_WRITE -= other.COUNT_WRITE
	this.COUNT_MISC -= other.COUNT_MISC

	this.SUM_TIMER_WAIT -= other.SUM_TIMER_WAIT
	this.SUM_TIMER_READ -= other.SUM_TIMER_READ
	this.SUM_TIMER_WRITE -= other.SUM_TIMER_WRITE
	this.SUM_TIMER_MISC -= other.SUM_TIMER_MISC

	this.SUM_NUMBER_OF_BYTES_READ -= other.SUM_NUMBER_OF_BYTES_READ
	this.SUM_NUMBER_OF_BYTES_WRITE -= other.SUM_NUMBER_OF_BYTES_WRITE
}

// return the totals of a slice of rows
func (t file_summary_by_instance_rows) totals() file_summary_by_instance_row {
	var totals file_summary_by_instance_row
	totals.FILE_NAME = "TOTALS"

	for i := range t {
		totals.add(t[i])
	}

	return totals
}

// clean up the given path reducing redundant stuff and return the clean path
func cleanup_path(path string) string {

	for {
		orig_path := path
		path = re_one_or_the_other.ReplaceAllString(path, "/")
		path = re_slash_dot_dot_slash.ReplaceAllString(path, "/")
		if orig_path == path { // no change so give up
			break
		}
	}

	return path
}

// From the original FILE_NAME we want to generate a simpler name to use.
// This simpler name may also merge several different filenames into one.
func (t file_summary_by_instance_row) simple_name(global_variables map[string]string) string {

	path := t.FILE_NAME

	if cached_result, err := cache.Get(path); err == nil {
		return cached_result
	}

	// @0024 --> $ (should do this more generically)
	path = re_dollar.ReplaceAllLiteralString(path, "$")

	// this should probably be ordered from most expected regexp to least
	if m1 := re_table_file.FindStringSubmatch(path); m1 != nil {
		// we may match temporary tables so check for them
		if m2 := re_temp_table.FindStringSubmatch(m1[2]); m2 != nil {
			return cache.Put(path, "<temp_table>")
		}

		// we may match partitioned tables so check for them
		if m3 := re_part_table.FindStringSubmatch(m1[2]); m3 != nil {
			return cache.Put(path, m1[1]+"."+m3[1]) // <schema>.<table> (less partition info)
		}

		return cache.Put(path, m1[1]+"."+m1[2]) // <schema>.<table>
	}
	if re_ibdata.MatchString(path) == true {
		return cache.Put(path, "<ibdata>")
	}
	if re_redo_log.MatchString(path) == true {
		return cache.Put(path, "<redo_log>")
	}
	if re_binlog.MatchString(path) == true {
		return cache.Put(path, "<binlog>")
	}
	if re_db_opt.MatchString(path) == true {
		return cache.Put(path, "<db_opt>")
	}
	if re_slowlog.MatchString(path) == true {
		return cache.Put(path, "<slow_log>")
	}
	if re_auto_cnf.MatchString(path) == true {
		return cache.Put(path, "<auto_cnf>")
	}
	// relay logs are a bit complicated. If a full path then easy to
	// identify,but if a relative path we may need to add $datadir,
	// but also if as I do we have a ../blah/somewhere/path then we
	// need to make it match too.
	if len(global_variables["relay_log"]) > 0 {
		relay_log := global_variables["relay_log"]
		if relay_log[0] != '/' { // relative path
			relay_log = cleanup_path(global_variables["datadir"] + relay_log) // datadir always ends in /
		}
		re_relay_log := relay_log + `\.(\d{6}|index)$`
		if regexp.MustCompile(re_relay_log).MatchString(path) == true {
			return cache.Put(path, "<relay_log>")
		}
	}
	if re_pid_file.MatchString(path) == true {
		return cache.Put(path, "<pid_file>")
	}
	if re_error_msg.MatchString(path) == true {
		return cache.Put(path, "<errmsg>")
	}
	if re_charset.MatchString(path) == true {
		return cache.Put(path, "<charset>")
	}
	return cache.Put(path, path)
}

// Convert the imported "table" to a merged one with merged data.
// Combine all entries with the same "FILE_NAME" by adding their values.
func merge_by_table_name(orig file_summary_by_instance_rows, global_variables map[string]string) file_summary_by_instance_rows {
	start := time.Now()
	t := make(file_summary_by_instance_rows, 0, len(orig))

	m := make(map[string]file_summary_by_instance_row)

	// iterate over source table
	for i := range orig {
		var file_name string
		var new_row file_summary_by_instance_row
		orig_row := orig[i]

		if orig_row.COUNT_STAR > 0 {
			file_name = orig_row.simple_name(global_variables)

			// check if we have an entry in the map
			if _, found := m[file_name]; found {
				new_row = m[file_name]
			} else {
				new_row.FILE_NAME = file_name
			}
			new_row.add(orig_row)
			m[file_name] = new_row // update the map with the new value
		}
	}

	// add the map contents back into the table
	for _, row := range m {
		t = append(t, row)
	}

	lib.Logger.Println("merge_by_table_name() took:", time.Duration(time.Since(start)).String())
	return t
}

// Select the raw data from the database into file_summary_by_instance_rows
// - filter out empty values
// - merge rows with the same name into a single row
// - change FILE_NAME into a more descriptive value.
func select_fsbi_rows(dbh *sql.DB) file_summary_by_instance_rows {
	var t file_summary_by_instance_rows
	start := time.Now()

	sql := "SELECT FILE_NAME, COUNT_STAR, SUM_TIMER_WAIT, COUNT_READ, SUM_TIMER_READ, SUM_NUMBER_OF_BYTES_READ, COUNT_WRITE, SUM_TIMER_WRITE, SUM_NUMBER_OF_BYTES_WRITE, COUNT_MISC, SUM_TIMER_MISC FROM file_summary_by_instance"

	rows, err := dbh.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var r file_summary_by_instance_row

		if err := rows.Scan(&r.FILE_NAME, &r.COUNT_STAR, &r.SUM_TIMER_WAIT, &r.COUNT_READ, &r.SUM_TIMER_READ, &r.SUM_NUMBER_OF_BYTES_READ, &r.COUNT_WRITE, &r.SUM_TIMER_WRITE, &r.SUM_NUMBER_OF_BYTES_WRITE, &r.COUNT_MISC, &r.SUM_TIMER_MISC); err != nil {
			log.Fatal(err)
		}
		t = append(t, r)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	lib.Logger.Println("select_fsbi_rows() took:", time.Duration(time.Since(start)).String())

	return t
}

// remove the initial values from those rows where there's a match
// - if we find a row we can't match ignore it
func (this *file_summary_by_instance_rows) subtract(initial file_summary_by_instance_rows) {
	i_by_name := make(map[string]int)

	// iterate over rows by name
	for i := range initial {
		i_by_name[initial[i].name()] = i
	}

	for i := range *this {
		if _, ok := i_by_name[(*this)[i].name()]; ok {
			initial_i := i_by_name[(*this)[i].name()]
			(*this)[i].subtract(initial[initial_i])
		}
	}
}

func (t file_summary_by_instance_rows) Len() int      { return len(t) }
func (t file_summary_by_instance_rows) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t file_summary_by_instance_rows) Less(i, j int) bool {
	return (t[i].SUM_TIMER_WAIT > t[j].SUM_TIMER_WAIT) ||
		((t[i].SUM_TIMER_WAIT == t[j].SUM_TIMER_WAIT) && (t[i].FILE_NAME < t[j].FILE_NAME))
}

func (t *file_summary_by_instance_rows) sort() {
	sort.Sort(t)
}

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
func (t file_summary_by_instance_rows) needs_refresh(t2 file_summary_by_instance_rows) bool {
	my_totals := t.totals()
	t2_totals := t2.totals()

	return my_totals.SUM_TIMER_WAIT > t2_totals.SUM_TIMER_WAIT
}
