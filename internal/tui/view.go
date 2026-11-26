package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/sjmudd/ps-top/v2/internal/presenter"
)

// minWidth and minHeight are the smallest screen drawn: smaller says so.
const (
	minWidth  = 20
	minHeight = 6
)

// View draws the screen on the alternate screen.
func (m *model) View() tea.View {
	v := tea.NewView(m.screen())
	v.AltScreen = true
	return v
}

// screen returns the screen's lines, joined, each padded or cut to the
// width so its background fills the line.
func (m *model) screen() string {
	w, h := m.width, m.height
	if w == 0 || h == 0 {
		return "" // size not known yet
	}
	if w < minWidth || h < minHeight {
		return fmt.Sprintf("terminal too small: %d by %d", w, h)
	}
	th := m.theme
	lines := make([]string, 0, h)
	lines = append(lines, th.top.Render(m.topLine(w)))

	// Lines 2 to H-1: help, or the table.
	body := h - 2
	if m.help {
		lines = append(lines, th.title.Render(fit("Help", w)))
		for _, l := range fill(helpLines(), body-1, "") {
			lines = append(lines, th.rows.Render(fit(l, w)))
		}
	} else if m.have && len(m.table.Parts) > 0 {
		lines = append(lines, th.title.Render(fit(m.table.Title, w)))
		messageRow := -1 // the middle row, while a message is shown
		if m.message != "" && m.now().Before(m.messageUntil) {
			messageRow = (body - 1) / 2
		}
		for i, r := range partLines(m.table.Parts, body-1) {
			if i == messageRow {
				lines = append(lines, m.messageLine(w))
				continue
			}
			lines = append(lines, th.rows.Render(fit(r, w)))
		}
	} else {
		var text []string
		if m.have {
			text = m.table.Lines(0)
		}
		title, headings, rows, totals := "", "", []string(nil), ""
		if len(text) > 0 {
			title = text[0]
		}
		if len(text) >= 3 {
			headings, rows, totals = text[1], text[2:len(text)-1], text[len(text)-1]
		}
		lines = append(lines, th.title.Render(fit(title, w)), th.headings.Render(fit(headings, w)))
		messageRow := -1 // the middle row, while a message is shown
		if m.message != "" && m.now().Before(m.messageUntil) {
			messageRow = (h - 5) / 2
		}
		blank := ""
		if m.have {
			blank = m.table.Blank()
		}
		for i, r := range fill(rows, h-5, blank) {
			if i == messageRow {
				lines = append(lines, m.messageLine(w))
				continue
			}
			lines = append(lines, th.rows.Render(fit(r, w)))
		}
		lines = append(lines, th.totals.Render(fit(totals, w)))
	}
	lines = append(lines, m.menuLine(w))
	return strings.Join(lines, "\n")
}

// topLine returns the top line: program, version, time, server and uptime,
// and on the right what the values cover, e.g. "[last 10s]".
func (m *model) topLine(w int) string {
	now := m.now()
	s := m.cfg.Server
	left := fmt.Sprintf("%s %s - %s %s:%s / %s, up %s", m.cfg.Program, m.cfg.Version, clock(now),
		s.Host, s.Port, s.Version, uptime(now.Sub(s.Started)))
	if !m.have || m.table.Style == "" {
		return fit(left, w)
	}
	right := "[" + m.table.Style + "]"
	if gap := w - runes(left) - runes(right); gap >= 1 {
		return left + strings.Repeat(" ", gap) + right
	}
	return fit(left, w)
}

// messageLine returns the message, centred, on the rows' background.
func (m *model) messageLine(w int) string {
	text := cut(" "+m.message+" ", w)
	left := (w - runes(text)) / 2
	right := w - left - runes(text)
	return m.theme.rows.Render(strings.Repeat(" ", left)) + m.theme.message.Render(text) +
		m.theme.rows.Render(strings.Repeat(" ", right))
}

// menuLine returns the menu: keys in [ ] in their own colour, as the old
// program draws it.
func (m *model) menuLine(w int) string {
	th := m.theme
	var b strings.Builder
	style, n := th.menu, 0
	for _, r := range cut(menuText(), w) {
		switch r {
		case '[':
			b.WriteString(th.menuBracket.Render("["))
			style = th.menuKey
		case ']':
			b.WriteString(th.menuBracket.Render("]"))
			style = th.menu
		default:
			b.WriteString(style.Render(string(r)))
		}
		n++
	}
	b.WriteString(th.menu.Render(strings.Repeat(" ", max(w-n, 0))))
	return b.String()
}

// fill returns lines with blank added, or lines cut, to n lines.
// partLines returns n lines showing the parts of a combined view: all but the
// last at the top, one under the other with a blank line between, and the
// last in a bottom section at the foot, below a blank line. If they do not
// fit, the top is cut first.
func partLines(parts []presenter.Table, n int) []string {
	var top []string
	for i, p := range parts[:len(parts)-1] {
		if i > 0 {
			top = append(top, "")
		}
		top = append(top, p.Lines(0)...)
	}
	bottom := parts[len(parts)-1].Lines(0)
	if len(parts) > 1 {
		bottom = append([]string{""}, bottom...)
	}
	if len(bottom) > n {
		bottom = bottom[len(bottom)-n:]
	}
	return append(fill(top, n-len(bottom), ""), bottom...)
}

func fill(lines []string, n int, blank string) []string {
	out := make([]string, n)
	for i := copy(out, lines); i < n; i++ {
		out[i] = blank
	}
	return out
}

// fit returns s cut or padded with spaces to w runes.
func fit(s string, w int) string {
	s = cut(s, w)
	return s + strings.Repeat(" ", w-runes(s))
}

// cut returns s limited to w runes.
func cut(s string, w int) string {
	if r := []rune(s); len(r) > w {
		return string(r[:w])
	}
	return s
}

func runes(s string) int { return len([]rune(s)) }

// secs returns d in seconds, e.g. "60s" or "0.5s".
func secs(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', -1, 64) + "s"
}

// clock returns the time of day as the old program shows it, e.g. " 0:43:26".
func clock(t time.Time) string {
	return fmt.Sprintf("%2d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

// uptime returns d as the old program shows the server's uptime, e.g.
// "33d 6h 15m 34s", "1m 0s", "36s".
func uptime(d time.Duration) string {
	secs := max(int(d/time.Second), 0)
	days, hours, minutes, seconds := secs/86400, secs/3600%24, secs/60%60, secs%60
	out := strconv.Itoa(seconds) + "s"
	if minutes > 0 || hours > 0 || days > 0 {
		out = strconv.Itoa(minutes) + "m " + out
	}
	if hours > 0 || days > 0 {
		out = strconv.Itoa(hours) + "h " + out
	}
	if days > 0 {
		out = strconv.Itoa(days) + "d " + out
	}
	return out
}
