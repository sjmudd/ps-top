// Package metric describes the shape of a collected metric: its value
// columns, how each behaves over time, and the metric types (current values
// or a difference) its views can ask for.
package metric

import (
	"errors"
	"fmt"
	"slices"
)

// Kind says how a value column behaves over time.
type Kind int

const (
	// Counter only increases. A decrease means the source was reset.
	Counter Kind = iota
	// Gauge may go up or down, e.g. memory currently in use.
	Gauge
)

// MetricType says which values a view of a metric shows, and so how many
// snapshots the collector sends for it.
type MetricType int

const (
	// CurrentValue shows the values of the latest snapshot as they are,
	// e.g. memory in use now. One snapshot is sent.
	CurrentValue MetricType = iota
	// DifferenceValues shows the change over a window: counters as their
	// increase, gauges as their latest value. Two snapshots are sent, the
	// window's start and end.
	DifferenceValues
)

func (t MetricType) String() string {
	switch t {
	case CurrentValue:
		return "CurrentValue"
	case DifferenceValues:
		return "DifferenceValues"
	}
	return fmt.Sprintf("MetricType(%d)", int(t))
}

// Name returns t's short name, as the command line and the metrics registry
// use it: "current" or "difference".
func (t MetricType) Name() string {
	switch t {
	case CurrentValue:
		return "current"
	case DifferenceValues:
		return "difference"
	}
	return t.String()
}

// ParseType returns the metric type with short name name.
func ParseType(name string) (MetricType, error) {
	for _, t := range []MetricType{CurrentValue, DifferenceValues} {
		if t.Name() == name {
			return t, nil
		}
	}
	return 0, fmt.Errorf("metric: unknown style %q, want current or difference", name)
}

// Snapshots returns how many snapshots are sent for t.
func (t MetricType) Snapshots() int {
	if t == DifferenceValues {
		return 2
	}
	return 1
}

// RingSize returns how many snapshots must be held to send t over a window
// of n collect intervals.
func (t MetricType) RingSize(n int) int {
	if t == DifferenceValues {
		return n + 1
	}
	return 1
}

// Merge says how the values of entries merged into one key are combined, and
// how a column's total over all entries is formed.
type Merge int

const (
	// Sum adds the values. The total is the sum over all entries.
	Sum Merge = iota
	// DistinctPart counts the distinct non-empty values of key part Part
	// among the entries merged, e.g. the hosts of a user's connections. The
	// entries' own values are not used. The total is the distinct count over
	// all entries, not the sum of the column: two users on one host are one
	// host.
	DistinctPart
)

// Column describes one value column. Name is the column's name in the
// performance_schema table, e.g. "SUM_TIMER_WAIT". If Expr is set, the value
// is that expression instead, read under Name, e.g. Expr "COUNT_ALLOC +
// COUNT_FREE" with Name "MEM_OPS". Merge says how values are combined when
// entries are merged (Sum unless set), and Part is the key part a
// DistinctPart column counts.
type Column struct {
	Name  string
	Kind  Kind
	Expr  string
	Merge Merge
	Part  int
}

// Definition describes a metric. Types are the metric types its views can
// ask for, the first being the default. ResetColumn is the index of the
// counter column whose total across all rows is used to detect a reset. It
// is used, and must be a counter, only if Types has DifferenceValues.
type Definition struct {
	Name        string
	Columns     []Column
	Types       []MetricType
	ResetColumn int
}

// Accepts reports whether t is one of d's types.
func (d Definition) Accepts(t MetricType) bool { return slices.Contains(d.Types, t) }

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
		switch c.Merge {
		case Sum:
		case DistinctPart:
			if c.Kind != Gauge {
				return fmt.Errorf("metric %q: distinct column %q must be a gauge", d.Name, c.Name)
			}
			if c.Part < 0 {
				return fmt.Errorf("metric %q: column %q has key part %d", d.Name, c.Name, c.Part)
			}
		default:
			return fmt.Errorf("metric %q: column %q has invalid merge %d", d.Name, c.Name, int(c.Merge))
		}
	}
	if len(d.Types) == 0 {
		return fmt.Errorf("metric %q: no types", d.Name)
	}
	for i, t := range d.Types {
		if t < CurrentValue || t > DifferenceValues {
			return fmt.Errorf("metric %q: invalid type %d", d.Name, int(t))
		}
		if slices.Contains(d.Types[:i], t) {
			return fmt.Errorf("metric %q: duplicate type %v", d.Name, t)
		}
	}
	if !d.Accepts(DifferenceValues) {
		return nil
	}
	if d.ResetColumn < 0 || d.ResetColumn >= len(d.Columns) {
		return fmt.Errorf("metric %q: reset column %d out of range", d.Name, d.ResetColumn)
	}
	if d.Columns[d.ResetColumn].Kind != Counter {
		return fmt.Errorf("metric %q: reset column %q is not a counter", d.Name, d.Columns[d.ResetColumn].Name)
	}
	return nil
}
