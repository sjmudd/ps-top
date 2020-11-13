// Package file_io contains the routines for
// managing the file_summary_by_instance table.
package file_io

import (
	"regexp"

	"github.com/sjmudd/ps-top/global"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/logger"
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

// Row contains a row from file_summary_by_instance
type Row struct {
	Name                  string
	CountStar             uint64
	CountRead             uint64
	CountWrite            uint64
	CountMisc             uint64
	SumTimerWait          uint64
	SumTimerRead          uint64
	SumTimerWrite         uint64
	SumTimerMisc          uint64
	SumNumberOfBytesRead  uint64
	SumNumberOfBytesWrite uint64
}

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
	reDoubleWrite      = regexp.MustCompile(`/#ib_[0-9_]+\.dblwr$`) // i1/#ib_16384_0.dblwr
	reIbdata           = regexp.MustCompile(`/ibdata\d+$`)
	reIbtmp            = regexp.MustCompile(`/ibtmp\d+$`)
	reRedoLog          = regexp.MustCompile(`/ib_logfile\d+$`)
	reUndoLog          = regexp.MustCompile(`/undo_\d+$`)
	reBinlog           = regexp.MustCompile(`/binlog\.(\d{6}|index)$`)
	reDbOpt            = regexp.MustCompile(`/db\.opt$`)
	reSlowlog          = regexp.MustCompile(`/slowlog$`)
	reAutoCnf          = regexp.MustCompile(`/auto\.cnf$`)
	rePidFile          = regexp.MustCompile(`/[^/]+\.pid$`)
	reErrorMsg         = regexp.MustCompile(`/share/[^/]+/errmsg\.sys$`)
	reCharset          = regexp.MustCompile(`/share/charsets/Index\.xml$`)
	reDollar           = regexp.MustCompile(`@0024`) // FIXME - add me to catch @0024 --> $ (specific case)
)

// Valid checks if the row is valid and if asked to do so logs the problem
func (row Row) Valid(logProblem bool) bool {
	var problem bool
	if (row.CountStar < row.CountRead) ||
		(row.CountStar < row.CountWrite) ||
		(row.CountStar < row.CountMisc) {
		problem = true
		if logProblem {
			logger.Println("Row.Valid() FAILED (count)", row)
		}
	}
	if (row.SumTimerWait < row.SumTimerRead) ||
		(row.SumTimerWait < row.SumTimerWrite) ||
		(row.SumTimerWait < row.SumTimerMisc) {
		problem = true
		if logProblem {
			logger.Println("Row.Valid() FAILED (sumTimer)", row)
		}
	}
	return problem
}

// Add rows together, keeping the name of first row
func add(row, other Row) Row {
	newRow := row

	newRow.CountStar += other.CountStar
	newRow.CountRead += other.CountRead
	newRow.CountWrite += other.CountWrite
	newRow.CountMisc += other.CountMisc

	newRow.SumTimerWait += other.SumTimerWait
	newRow.SumTimerRead += other.SumTimerRead
	newRow.SumTimerWrite += other.SumTimerWrite
	newRow.SumTimerMisc += other.SumTimerMisc

	newRow.SumNumberOfBytesRead += other.SumNumberOfBytesRead
	newRow.SumNumberOfBytesWrite += other.SumNumberOfBytesWrite

	return newRow
}

// sometimes the values can drop and catch us out. This routines hides that by returning 0
func validSubtract(this, that uint64) uint64 {
	if this > that {
		return this - that
	}
	return 0
}

// subtract one set of values from another one keeping the original row name
// - use validSubtract() to catch negative jumps which do happen from time to time.
func subtract(row, other Row) Row {
	newRow := row

	newRow.CountStar = validSubtract(row.CountStar, other.CountStar)
	newRow.CountRead = validSubtract(row.CountRead, other.CountRead)
	newRow.CountWrite = validSubtract(row.CountWrite, other.CountWrite)
	newRow.CountMisc = validSubtract(row.CountMisc, other.CountMisc)

	newRow.SumTimerWait = validSubtract(row.SumTimerWait, other.SumTimerWait)
	newRow.SumTimerRead = validSubtract(row.SumTimerRead, other.SumTimerRead)
	newRow.SumTimerWrite = validSubtract(row.SumTimerWrite, other.SumTimerWrite)
	newRow.SumTimerMisc = validSubtract(row.SumTimerMisc, other.SumTimerMisc)

	newRow.SumNumberOfBytesRead = validSubtract(row.SumNumberOfBytesRead, other.SumNumberOfBytesRead)
	newRow.SumNumberOfBytesWrite = validSubtract(row.SumNumberOfBytesWrite, other.SumNumberOfBytesWrite)

	return newRow
}

// simplifyName converts the filename into something easier to
// recognise.  This simpler name may also merge several different
// filenames into one.  To help with performance the path replacements
// are stored in a cache so they can be used again on the next run.
func (row Row) simplifyName(globalVariables *global.Variables) string {
	path := row.Name

	if cachedResult, err := cache.get(path); err == nil {
		return cachedResult
	}

	// @0024 --> $ (should do this more generically)
	path = reDollar.ReplaceAllLiteralString(path, "$")

	// this should probably be ordered from most expected regexp to least
	if m1 := reTableFile.FindStringSubmatch(path); m1 != nil {
		// we may match temporary tables so check for them
		if m2 := reTempTable.FindStringSubmatch(m1[2]); m2 != nil {
			return cache.put(path, "<temp_table>")
		}

		// we may match partitioned tables so check for them
		if m3 := rePartTable.FindStringSubmatch(m1[2]); m3 != nil {
			return cache.put(path, lib.TableName(m1[1], m3[1])) // <schema>.<table> (less partition info)
		}

		return cache.put(path, rc.Munge(lib.TableName(m1[1], m1[2]))) // <schema>.<table>
	}
	if reIbtmp.MatchString(path) {
		return cache.put(path, "<ibtmp>")
	}
	if reIbdata.MatchString(path) {
		return cache.put(path, "<ibdata>")
	}
	if reUndoLog.MatchString(path) {
		return cache.put(path, "<undo_log>")
	}
	if reRedoLog.MatchString(path) {
		return cache.put(path, "<redo_log>")
	}
	if reDoubleWrite.MatchString(path) {
		return cache.put(path, "<doublewrite>")
	}
	if reBinlog.MatchString(path) {
		return cache.put(path, "<binlog>")
	}
	if reDbOpt.MatchString(path) {
		return cache.put(path, "<db_opt>")
	}
	if reSlowlog.MatchString(path) {
		return cache.put(path, "<slow_log>")
	}
	if reAutoCnf.MatchString(path) {
		return cache.put(path, "<auto_cnf>")
	}
	// relay logs are a bit complicated. If a full path then easy to
	// identify, but if a relative path we may need to add $datadir,
	// but also if as I do we have a ../blah/somewhere/path then we
	// need to make it match too.
	if len(globalVariables.Get("relay_log")) > 0 {
		relayLog := globalVariables.Get("relay_log")
		if relayLog[0] != '/' { // relative path
			relayLog = cleanupPath(globalVariables.Get("datadir") + relayLog) // datadir always ends in /
		}
		reRelayLog := relayLog + `\.(\d{6}|index)$`
		if regexp.MustCompile(reRelayLog).MatchString(path) {
			return cache.put(path, "<relay_log>")
		}
	}
	if rePidFile.MatchString(path) {
		return cache.put(path, "<pid_file>")
	}
	if reErrorMsg.MatchString(path) {
		return cache.put(path, "<errmsg>")
	}
	if reCharset.MatchString(path) {
		return cache.put(path, "<charset>")
	}
	// clean up datadir to <datadir>
	if len(globalVariables.Get("datadir")) > 0 {
		reDatadir := regexp.MustCompile("^" + globalVariables.Get("datadir"))
		path = reDatadir.ReplaceAllLiteralString(path, "<datadir>/")
	}

	return cache.put(path, path)
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

// HasData indicates if there is data in the row (for counting valid rows)
func (row *Row) HasData() bool {
	return row != nil && row.SumTimerWait > 0
}
