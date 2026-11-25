package tui

import (
	"image/color"

	lipgloss "charm.land/lipgloss/v2"
)

// theme is the style of each screen area.
type theme struct {
	top, title, headings, rows, totals lipgloss.Style
	menu, menuBracket, menuKey         lipgloss.Style
	message                            lipgloss.Style
}

// oldTheme is the old program's colours (display/display.go), and messages
// grey on light red.
func oldTheme() theme {
	on := func(fg, bg color.Color) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(fg).Background(bg)
	}
	return theme{
		top:         on(lipgloss.Black, lipgloss.BrightBlack),
		title:       on(lipgloss.Black, lipgloss.Cyan),
		headings:    on(lipgloss.BrightWhite, lipgloss.Black),
		rows:        on(lipgloss.BrightBlack, lipgloss.Black),
		totals:      on(lipgloss.BrightWhite, lipgloss.Black),
		menu:        on(lipgloss.Black, lipgloss.BrightBlack),
		menuBracket: on(lipgloss.Black, lipgloss.BrightBlack),
		menuKey:     on(lipgloss.Color("#8b0000"), lipgloss.BrightBlack),
		message:     on(lipgloss.White, lipgloss.BrightRed),
	}
}
