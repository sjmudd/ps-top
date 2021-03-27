// Package file_io contains the routines for
// managing the file_summary_by_instance table.
package file_io

import (
	"regexp"

	"github.com/sjmudd/ps-top/global"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/rc"
)

// patternList for regexp replacements
type patternList struct {
	re          *regexp.Regexp
	replacement string
}

//     foo/../bar --> foo/bar   perl: $new =~ s{[^/]+/\.\./}{/};
//     /./        --> /         perl: $new =~ s{/\./}{};
//     //         --> /         perl: $new =~ s{//}{/};

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

	patterns = []patternList{
		{reIbtmp, "<ibtmp>"},
		{reIbdata, "<ibdata>"},
		{reUndoLog, "<undo_log>"},
		{reRedoLog, "<redo_log>"},
		{reDoubleWrite, "<doublewrite>"},
		{reBinlog, "<binlog>"},
		{reDbOpt, "<db_opt>"},
		{reSlowlog, "<slow_log>"},
		{reAutoCnf, "<auto_cnf>"},
		{rePidFile, "<pid_file>"},
		{reErrorMsg, "<errmsg>"},
		{reCharset, "<charset>"},
	}
)

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

// provide a regexp match list and return the replacement if the path
// matches one of the patterns
func matchPattern(patterns []patternList, path string) (string, bool) {
	for _, pattern := range patterns {
		if pattern.re.MatchString(path) {
			return pattern.replacement, true
		}
	}
	return "", false
}

// simplify converts the filename into something easier to
// recognise.  This simpler name may also merge several different
// filenames into one.  To help with performance the path replacements
// are stored in a cache so they can be used again on the next run.
func simplify(path string, globalVariables *global.Variables) string {
	if cachedResult, err := cache.get(path); err == nil {
		return cachedResult
	}

	return cache.put(path, uncachedSimplify(path, globalVariables))
}

// generic interface to make testing easier
type getter interface {
	Get(string) string
}

// uncachedSimplify converts the filename into something easier to
// recognise.  This simpler name may also merge several different
// filenames into one.
func uncachedSimplify(path string, globalVariables getter) string {
	// @0024 --> $ (should do this more generically)
	path = reDollar.ReplaceAllLiteralString(path, "$")

	// this should probably be ordered from most expected regexp to least
	if m1 := reTableFile.FindStringSubmatch(path); m1 != nil {
		// we may match temporary tables so check for them
		if m2 := reTempTable.FindStringSubmatch(m1[2]); m2 != nil {
			return "<temp_table>"
		}

		// we may match partitioned tables so check for them
		if m3 := rePartTable.FindStringSubmatch(m1[2]); m3 != nil {
			return lib.QualifiedTableName(m1[1], m3[1]) // <schema>.<table> (less partition info)
		}

		return rc.Munge(lib.QualifiedTableName(m1[1], m1[2])) // <schema>.<table>
	}

	// bulk match various values
	if replacement, found := matchPattern(patterns, path); found {
		return replacement
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
			return "<relay_log>"
		}
	}
	// clean up datadir to <datadir>
	if len(globalVariables.Get("datadir")) > 0 {
		reDatadir := regexp.MustCompile("^" + globalVariables.Get("datadir"))
		path = reDatadir.ReplaceAllLiteralString(path, "<datadir>/")
	}

	return path
}
