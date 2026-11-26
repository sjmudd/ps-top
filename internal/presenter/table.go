package presenter

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Table is a view as cells, as View.Table returns it: what each cell shows,
// formatted and aligned, and its role, so the printer (Lines) and the TUI
// (colours by role) show the same text.
type Table struct {
	// Title is the line above the headings, e.g. "Table I/O Latency
	// (table_io_waits_summary_by_table) 8 rows, last 10s".
	Title string
	// Style is what the values cover, as the title ends: "last 10s" or
	// "current". Empty while waiting.
	Style string
	// Groups, if set, are titles for the groups of Headings, "" for none,
	// shown centred on a line above them.
	Groups   []string
	Headings []TableGroup
	Rows     [][]TableGroup // one per row shown, in order
	Totals   []TableGroup   // nil: no totals line
	// Parts, if set, are the tables of a Combined view, shown one under the
	// other below Title, in place of headings, rows and totals.
	Parts []Table
	// Waiting is set when the collector has too few snapshots yet: the
	// title says so and there are no headings, rows or totals.
	Waiting bool
}

// TableGroup is the cells between two "|". Cells in a group are separated
// by a space.
type TableGroup []TableCell

// TableCell is one cell of a Table: its text, padded to the cell's width,
// and its role. (Cell is a cell of a view's definition.)
type TableCell struct {
	Text string
	Role Role
}

// Role says what a cell shows.
type Role int

const (
	RoleHeading Role = iota // a column heading
	RoleValue               // a formatted value, e.g. "  8.00 ms"
	RoleShare               // a share, e.g. " 40.0%"
	RoleKey                 // the row's key, or "Totals" on the totals row
)

// Lines returns the table as text lines, each cut at width runes (0: not
// cut) with trailing spaces trimmed: the title, the group titles if any, the
// headings, one line per row and the totals if any. While waiting, the
// title only. With Parts: the title, then each part's lines, a blank line
// between parts.
func (t Table) Lines(width int) []string {
	lines := []string{t.Title}
	if len(t.Parts) > 0 {
		for i, p := range t.Parts {
			if i > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, p.Lines(0)...)
		}
	} else if !t.Waiting {
		if len(t.Groups) > 0 {
			lines = append(lines, t.groupTitles())
		}
		lines = append(lines, groupsText(t.Headings))
		for _, r := range t.Rows {
			lines = append(lines, groupsText(r))
		}
		if t.Totals != nil {
			lines = append(lines, groupsText(t.Totals))
		}
	}
	for i := range lines {
		lines[i] = strings.TrimRight(cut(lines[i], width), " ")
	}
	return lines
}

// Blank returns an empty row: the headings line with each character but "|"
// a space, and trailing spaces trimmed. The TUI fills unused rows with it,
// as v1 does. Empty while waiting.
func (t Table) Blank() string {
	if t.Waiting {
		return ""
	}
	blank := strings.Map(func(r rune) rune {
		if r == '|' {
			return r
		}
		return ' '
	}, groupsText(t.Headings))
	return strings.TrimRight(blank, " ")
}

// groupTitles returns the line of group titles: each centred in its heading
// group's width, separated by "|", trailing spaces trimmed. A title longer
// than its group is cut.
func (t Table) groupTitles() string {
	titles := make([]string, len(t.Headings))
	for g, h := range t.Headings {
		w := utf8.RuneCountInString(groupsText([]TableGroup{h}))
		var s string
		if g < len(t.Groups) {
			s = cut(t.Groups[g], w)
		}
		left := (w - utf8.RuneCountInString(s)) / 2
		titles[g] = fmt.Sprintf("%*s%-*s", left, "", w-left, s)
	}
	return strings.TrimRight(strings.Join(titles, "|"), " ")
}

// groupsText joins groups with "|", and the cells of a group with a space.
func groupsText(groups []TableGroup) string {
	var b strings.Builder
	for g, cells := range groups {
		if g > 0 {
			b.WriteString("|")
		}
		for i, c := range cells {
			if i > 0 {
				b.WriteString(" ")
			}
			b.WriteString(c.Text)
		}
	}
	return b.String()
}
