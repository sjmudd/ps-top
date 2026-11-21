// Package metric describes the shape of a collected metric: its value columns
// and how each behaves over time.
package metric

import (
	"errors"
	"fmt"
)

// Kind says how a value column behaves over time.
type Kind int

const (
	// Counter only increases. A decrease means the source was reset.
	Counter Kind = iota
	// Gauge may go up or down, e.g. memory currently in use.
	Gauge
)

// Column describes one value column. Name is the column's name in the
// performance_schema table, e.g. "SUM_TIMER_WAIT".
type Column struct {
	Name string
	Kind Kind
}

// Definition describes a metric. ResetColumn is the index of the counter
// column whose total across all rows is used to detect a reset.
type Definition struct {
	Name        string
	Columns     []Column
	ResetColumn int
}

// Validate checks the definition is usable.
func (d Definition) Validate() error {
	if d.Name == "" {
		return errors.New("metric: definition has no name")
	}
	if len(d.Columns) == 0 {
		return fmt.Errorf("metric %q: no columns", d.Name)
	}
	seen := make(map[string]bool, len(d.Columns))
	for i, c := range d.Columns {
		if c.Name == "" {
			return fmt.Errorf("metric %q: column %d has no name", d.Name, i)
		}
		if seen[c.Name] {
			return fmt.Errorf("metric %q: duplicate column %q", d.Name, c.Name)
		}
		seen[c.Name] = true
		if c.Kind < Counter || c.Kind > Gauge {
			return fmt.Errorf("metric %q: column %q has invalid kind %d", d.Name, c.Name, int(c.Kind))
		}
	}
	if d.ResetColumn < 0 || d.ResetColumn >= len(d.Columns) {
		return fmt.Errorf("metric %q: reset column %d out of range", d.Name, d.ResetColumn)
	}
	if d.Columns[d.ResetColumn].Kind != Counter {
		return fmt.Errorf("metric %q: reset column %q is not a counter", d.Name, d.Columns[d.ResetColumn].Name)
	}
	return nil
}
