// Package rc provides routines to read ~/.pstoprc
// ps-top / ps-stats configuration
// - and to munge some table names based on the [munge] section (if present)
package rc

import (
	"os"
	"regexp"

	go_ini "github.com/vaughan0/go-ini" // not sure what to do with dashes in names

	"github.com/sjmudd/ps-top/log"
)

const (
	pstoprc = "~/.pstoprc" // location of the default pstop config file
)

// A single regexp expression from ~/.pstoprc
type mungeRegexp struct {
	pattern string
	replace string // static string replacement
	re      *regexp.Regexp
	valid   bool
}

var (
	haveRegexps bool // Do we have any valid data? We don't check yet if it's valid.
	regexps     []mungeRegexp
	loaded      bool // not concurrency safe, but not needed yet!
)

// modifyFilename replaces ~ with contents of HOME environment variable
func modifyFilename(filename string) string {
	for i := range filename {
		if filename[i] == '~' {
			filename = filename[:i] + os.Getenv("HOME") + filename[i+1:]
			break
		}
	}

	return filename
}

// Load the ~/.pstoprc regexp expressions in section [munge]
func loadRegexps() {
	haveRegexps = false
	filename := modifyFilename(pstoprc)

	// Is the file there? If not it is not fatal and we just return.
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	// If we get here the file is readable, so close it again.
	if err = f.Close(); err != nil {
		log.Printf("loadRegexps: f.Close() failed: %v", err)
	}

	// Load and process the ini file.
	i, err := go_ini.LoadFile(filename)
	if err != nil {
		log.Fatalf("Could not load %q: %v", filename, err)
	}

	// Note: This is wrong if I want to have an _ordered_ list of regexps
	// as go-ini provides me a hash so I lose the ordering. This may not
	// be desirable but as a first step accept this is broken.
	section := i.Section("munge")

	regexps = make([]mungeRegexp, 0, len(section))

	// look for regexps and load them in...
	for k, v := range section {
		addPattern(k, v)
	}

	// no check yet for validity of input data, just that we have regexp filters.
	if len(regexps) > 0 {
		log.Printf("found %d regexps to use to munge output", len(regexps))
	}
}

// addPattern adds the regexp and replacement pattern to our list.
func addPattern(pattern string, replace string) {
	var err error

	m := mungeRegexp{
		pattern: pattern,
		replace: replace,
	}

	m.re, err = regexp.Compile(m.pattern)
	if err == nil {
		m.valid = true
	}

	regexps = append(regexps, m)
	haveRegexps = true
}

// Munge Optionally munges table names so they can be combined.
// - this reads ~/.pstoprc for configuration information.
// - e.g.
// [munge]
// <re_match> = <replace>
// _[0-9]{8}$ = _YYYYMMDD
// _[0-9]{6}$ = _YYYYMM
func Munge(name string) string {
	// lazy loading of regexp expressions when needed
	if !loaded {
		loadRegexps()
		loaded = true
	}
	if !haveRegexps {
		return name // nothing to do so return what we were given.
	}

	munged := name

	for i := range regexps {
		if regexps[i].valid {
			if regexps[i].re.MatchString(munged) {
				munged = regexps[i].re.ReplaceAllLiteralString(munged, regexps[i].replace)
			}
		}
	}

	return munged
}
