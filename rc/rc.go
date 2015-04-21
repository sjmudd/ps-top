// Routines to read ~/.pstoprc [pstop configuration]
// - and to munge some table names based on the [munge] section (if present)
package rc

import (
	"log"
	"os"
	"regexp"
	"strings"

	go_ini "github.com/vaughan0/go-ini" // not sure what to do with dashes in names
)

const (
	pstoprc = "~/.pstoprc"	// location of the default config file
)

// holds a single regexp expression from ~/.pstoprc
type munge_regexp struct {
	pattern string
	replace string
	re      *regexp.Regexp
	valid   bool
}

// holds a slice of regexp expressions
type munge_regexps []munge_regexp

var (
	regexps        munge_regexps
	loaded_regexps bool		// have we [attempted to] loaded data?
	have_regexps   bool		// do we have any valid data?
)

// There must be a better way of doing this. Fix me...
// Copied from github.com/sjmudd/mysql_defaults_file so I should share this common code or fix it.
// Return the environment value of a given name.
func get_environ(name string) string {
	for i := range os.Environ() {
		s := os.Environ()[i]
		k_v := strings.Split(s, "=")

		if k_v[0] == name {
			return k_v[1]
		}
	}
	return ""
}

// convert ~ to $HOME
// Copied from github.com/sjmudd/mysql_defaults_file so I should share this common code or fix it.
func convert_filename(filename string) string {
	for i := range filename {
		if filename[i] == '~' {
			//                      fmt.Println("Filename before", filename )
			filename = filename[:i] + get_environ("HOME") + filename[i+1:]
			//                      fmt.Println("Filename after", filename )
			break
		}
	}

	return filename
}

// load the data from pstop and store it in a global variable
func load_regexps() {
	var filename string

	if loaded_regexps {
		return
	}

	have_regexps = false
	filename = convert_filename(filename)

	// check if the file is there

	f, err := os.Open(filename)
	if err != nil {
		return // can't open file. This is not fatal. We just can't do anything useful.
	}
	// if we get here the file is readable, so close it again.
	err = f.Close()
	if err != nil {
	}

	// open with go_ini
	i, err := go_ini.LoadFile(filename)
	if err != nil {
		log.Fatal("Could not load ~/.pstoprc", filename, ":", err)
	}

	// note: this is wrong if I want to have an _ordered_ list of regexps
	// as go-ini provides me a hash so I lose the ordering. This may not
	// be desirable but as a first step accept this is broken.
	section := i.Section("munge")

	// make a map to put some data in
	regexps = make(munge_regexps, 0, len(section))

	// now look for regexps and load them in...
	for k, v := range section {
		var m munge_regexp
		m.pattern = k
		m.replace = v
		var err error
		m.re, err = regexp.Compile(m.pattern)
		if err == nil {
			m.valid = true
		}
		regexps = append(regexps, m)
	}

	loaded_regexps = true // remember we've loaded data
	if len(regexps) > 0 {
		have_regexps = true
	}
}

// Optionally munge table names so they can be combined.
// - this reads ~/.pstoprc for configuration information.
// - e.g.
// [munge]
// <re_match> = <replace>
// _[0-9]{8}$ = _YYYYMMDD
// _[0-9]{6}$ = _YYYYMM
func Munge(name string) string {
	if !loaded_regexps {
		load_regexps()
	}
	if !have_regexps {
		return name // nothing to do so return what we were given.
	}

	munged := name

	//	for r in range  ... {
	//		if r.valid {
	//			if r.re.MatchString(munged) {
	//				munged = r.re.ReplaceAllLiteralString(munged, r.replace)
	//			}
	//		}
	//	}

	return munged
}
