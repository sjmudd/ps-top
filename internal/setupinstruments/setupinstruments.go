// Package setupinstruments makes sure the performance_schema instruments some
// views need are enabled and timed, and puts back what it changed.
//
// For each pattern, the rows of setup_instruments with ENABLED or TIMED set to
// NO are saved and then set to YES, YES. Restore sets each saved row back to
// its saved values. If any pattern has such rows and the table cannot be
// updated, Enable fails: the views would not show what they should.
package setupinstruments

import (
	"context"
	"database/sql"
	"fmt"
)

// Table is the table Enable updates.
const Table = "performance_schema.setup_instruments"

// Patterns are the instruments, as LIKE patterns, that the mutex and stage
// views need.
var Patterns = []string{"wait/synch/mutex/%", "stage/sql/%"}

const (
	selectSQL = "SELECT NAME, ENABLED, TIMED FROM " + Table + " WHERE NAME LIKE ? AND 'NO' IN (ENABLED, TIMED)"
	updateSQL = "UPDATE " + Table + " SET ENABLED = ?, TIMED = ? WHERE NAME = ?"
)

// row is one instrument's settings.
type row struct{ name, enabled, timed string }

// Saved holds the settings of the instruments Enable changed.
type Saved struct {
	db   *sql.DB
	rows []row
}

// Enable enables and times every instrument matching patterns. canUpdate says
// whether the table may be updated. Nothing is changed unless every pattern
// can be fully enabled.
func Enable(ctx context.Context, db *sql.DB, patterns []string, canUpdate bool) (*Saved, error) {
	var todo []row
	for _, p := range patterns {
		rows, err := disabled(ctx, db, p)
		if err != nil {
			return nil, err
		}
		if len(rows) > 0 && !canUpdate {
			return nil, fmt.Errorf("setupinstruments: '%s' is not fully enabled and setup_instruments cannot be updated", p)
		}
		todo = append(todo, rows...)
	}
	s := &Saved{db: db}
	for _, r := range todo {
		if _, err := db.ExecContext(ctx, updateSQL, "YES", "YES", r.name); err != nil {
			err = fmt.Errorf("setupinstruments: enable %s: %w", r.name, err)
			if rerr := s.Restore(context.WithoutCancel(ctx)); rerr != nil {
				return nil, fmt.Errorf("%w; %w", err, rerr)
			}
			return nil, err
		}
		s.rows = append(s.rows, r)
	}
	return s, nil
}

// Changed returns how many instruments Enable changed.
func (s *Saved) Changed() int { return len(s.rows) }

// Restore sets every changed instrument back to its saved settings.
func (s *Saved) Restore(ctx context.Context) error {
	for len(s.rows) > 0 {
		r := s.rows[len(s.rows)-1]
		if _, err := s.db.ExecContext(ctx, updateSQL, r.enabled, r.timed, r.name); err != nil {
			return fmt.Errorf("setupinstruments: restore %s: %w", r.name, err)
		}
		s.rows = s.rows[:len(s.rows)-1]
	}
	return nil
}

// disabled returns the instruments matching pattern that are not enabled or
// not timed.
func disabled(ctx context.Context, db *sql.DB, pattern string) ([]row, error) {
	rs, err := db.QueryContext(ctx, selectSQL, pattern)
	if err != nil {
		return nil, fmt.Errorf("setupinstruments: %s: %w", pattern, err)
	}
	defer func() { _ = rs.Close() }()
	var out []row
	for rs.Next() {
		var r row
		if err := rs.Scan(&r.name, &r.enabled, &r.timed); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rs.Err()
}
