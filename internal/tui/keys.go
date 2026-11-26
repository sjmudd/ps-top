package tui

import "strings"

// action is what a key does.
type action int

const (
	actQuit action = iota
	actSuspend
	actNext
	actPrev
	actHelp
	actIntervalUp
	actIntervalDown
	actWindowDown
	actWindowUp
	actStyle
	actAnonymise
	actReset
)

// binding is one entry of the key map. The menu line and the help screen are
// built from the key map, so they always match the keys.
type binding struct {
	keys  []string // as tea.KeyPressMsg.String() gives them, e.g. "ctrl+c"
	label string   // the keys as the help screen shows them
	menu  string   // on the menu line, the key in [ ], e.g. "[h]elp"; empty: not on the menu
	help  string   // what the help screen says it does
	act   action
}

// bindings is the key map, in menu order.
var bindings = []binding{
	{[]string{"+"}, "+", "[+-] Interval", "redraw less often: interval 1 s longer", actIntervalUp},
	{[]string{"-"}, "-", "", "redraw more often: interval 1 s shorter, down to the collect interval", actIntervalDown},
	{[]string{"shift+tab", "left"}, "Shift-Tab, Left", "[<] Prev", "previous view", actPrev},
	{[]string{"tab", "right"}, "Tab, Right", "[>] Next", "next view", actNext},
	{[]string{"w"}, "w", "[w/W] Window", "shorter window, in steps down to 5 s", actWindowDown},
	{[]string{"W"}, "W", "", "longer window, in steps up to 300 s", actWindowUp},
	{[]string{"r"}, "r", "[r] Style", "difference over the window, or current values", actStyle},
	{[]string{"a"}, "a", "[a] Anonymise", "anonymise schema, table and user names, on or off", actAnonymise},
	{[]string{"z"}, "z", "[z] Reset", "reset: windows start again from now", actReset},
	{[]string{"h", "?"}, "h, ?", "[h]elp", "this help, on or off", actHelp},
	{[]string{"q", "esc", "ctrl+c"}, "q, Esc, Ctrl-C", "[q]uit", "quit", actQuit},
	{[]string{"ctrl+z"}, "Ctrl-Z", "", "suspend: fg returns to ps-top", actSuspend},
}

// lookup returns what key does, and false if it does nothing.
func lookup(key string) (action, bool) {
	for _, b := range bindings {
		for _, k := range b.keys {
			if k == key {
				return b.act, true
			}
		}
	}
	return 0, false
}

// menuText returns the menu line's text, entries separated by two spaces.
func menuText() string {
	var entries []string
	for _, b := range bindings {
		if b.menu != "" {
			entries = append(entries, b.menu)
		}
	}
	return strings.Join(entries, "  ")
}

// helpLines returns the help screen's lines.
func helpLines() []string {
	lines := []string{"", "Keys:"}
	width := 0
	for _, b := range bindings {
		width = max(width, len(b.label))
	}
	for _, b := range bindings {
		lines = append(lines, "   "+b.label+strings.Repeat(" ", width-len(b.label))+"  "+b.help)
	}
	return append(lines, "", "Press h to return to the view")
}
