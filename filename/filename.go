// Package filename contains the routines for converting a filename to a MySQL object name.
package filename

import (
	"regexp"
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
	reTempTable2       = regexp.MustCompile(`#innodb_temp/temp_[0-9]+.ibt$`)
	rePartTable        = regexp.MustCompile(`(.+)#P#p(\d+|MAX)`)
	reDoubleWrite      = regexp.MustCompile(`/#ib_[0-9_]+\.dblwr$`) // i1/#ib_16384_0.dblwr
	reIbdata           = regexp.MustCompile(`/ibdata\d+$`)
	reIbtmp            = regexp.MustCompile(`/ibtmp\d+$`)
	reRedoLog          = regexp.MustCompile(`/(ib_logfile|#innodb_redo/#ib_redo)\d+$`)
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

// Simplify converts the filename into a more recognisable MySQL object name.
// This simpler name may also merge several different filenames into one.
// Values used here are cached for performance reasons.
func Simplify(path string, munger Munger, qualifiedNamer QualifiedNamer, datadir string, relaylog string) string {
	if cachedResult, err := cache.get(path); err == nil {
		return cachedResult
	}

	return cache.put(path, uncachedSimplify(path, munger, qualifiedNamer, datadir, relaylog))
}

// The Config interface is to pull out a value given a config setting
type Config interface {
	Get(setting string) string
}

// Munger allows us to modify the input string in any form we like, e.g. anonymising
type Munger func(string) string

// QualifiedNamer allows us to convert a schema and table into a single name
type QualifiedNamer func(string, string) string

// uncachedSimplify converts the filename into something easier to
// recognise.  This simpler name may also merge several different
// filenames into one.
func uncachedSimplify(path string, munger Munger, qualifiedNamer QualifiedNamer, datadir string, relaylog string) string {
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
			return munger(qualifiedNamer(m1[1], m3[1])) // <schema>.<table> (less partition info)
		}

		return munger(qualifiedNamer(m1[1], m1[2])) // <schema>.<table>
	}

	// catch 2nd temporary table format
	if m1 := reTempTable2.FindStringSubmatch(path); m1 != nil {
		return "<temp_table>"
	}

	// bulk match various values
	if replacement, found := matchPattern(patterns, path); found {
		return replacement
	}

	// relay logs are a bit complicated. If a full path then easy to
	// identify, but if a relative path we may need to add $datadir,
	// but also if as I do we have a ../blah/somewhere/path then we
	// need to make it match too.
	if len(relaylog) > 0 {
		if relaylog[0] != '/' { // relative path
			relaylog = cleanupPath(datadir + relaylog) // datadir always ends in /
		}
		reRelayLog := relaylog + `\.(\d{6}|index)$`
		if regexp.MustCompile(reRelayLog).MatchString(path) {
			return "<relay_log>"
		}
	}
	// clean up datadir to <datadir>
	if len(datadir) > 0 {
		reDatadir := regexp.MustCompile("^" + datadir)
		path = reDatadir.ReplaceAllLiteralString(path, "<datadir>/")
	}

	return path
}
