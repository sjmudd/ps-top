// Package munge renames table names by the rules of the [munge] section of
// ~/.pstoprc, as v1 does, so that tables named alike show as
// one row. Each line is a regular expression and its replacement:
//
//	[munge]
//	_[0-9]{8}$ = _YYYYMMDD
//	_[0-9]{6}$ = _YYYYMM
//
// shows shop.orders_20260101 and shop.orders_20260102 as
// shop.orders_YYYYMMDD.
//
// Known bug, kept from v1: the ini reader returns a section as
// a map, so the rules are applied in no fixed order, not the file's. Rules
// that can both match a name may give different results from run to run.
package munge

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"

	ini "github.com/vaughan0/go-ini"

	"github.com/sjmudd/ps-top/v2/internal/logging"
)

// FileName is the rules file in the home directory.
const FileName = ".pstoprc"

// rule is one line of the [munge] section.
type rule struct {
	re      *regexp.Regexp
	replace string // used literally
}

// Rules are the rules of a [munge] section. A nil Rules renames nothing.
type Rules []rule

// DefaultPath returns ~/.pstoprc, or "" if the home directory is unknown.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, FileName)
}

// Load returns the rules of path's [munge] section. A missing file, or an
// empty path, gives no rules. A file that cannot be read or parsed is an
// error. A regular expression that does not compile is skipped, as v1
// does.
func Load(path string) (Rules, error) {
	if path == "" {
		return nil, nil
	}
	f, err := ini.LoadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("munge: %s: %w", path, err)
	}
	var rules Rules
	for pattern, replace := range f.Section("munge") { // map order: see the package doc
		re, err := regexp.Compile(pattern)
		if err != nil {
			slog.Warn("skipped munge rule: invalid regexp", "path", path, "pattern", pattern, logging.KeyErr, err)
			continue
		}
		rules = append(rules, rule{re: re, replace: replace})
	}
	return rules, nil
}

// Apply returns name with each rule that matches it applied in turn.
func (r Rules) Apply(name string) string {
	for _, x := range r {
		name = x.re.ReplaceAllLiteralString(name, x.replace)
	}
	return name
}
