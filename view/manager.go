package view

import (
	"github.com/sjmudd/ps-top/display"
	"github.com/sjmudd/ps-top/pstable"
)

// TablerUpdater knows how to update the current tabler.
// Implemented by DBCollector to receive tabler updates when the view changes.
type TablerUpdater interface {
	SetCurrentTabler(pstable.Tabler)
}

// Manager manages view state, tabler selection, and display.
// It owns the current view, maps views to tablers, and handles rendering.
type Manager struct {
	view    View
	tablers map[Code]pstable.Tabler
	display *display.Display
	help    bool
	updater TablerUpdater
}

// NewManager creates a Manager with the given initial view, tabler mapping, and display.
// The updater is notified when the current tabler changes (including at initialization).
func NewManager(v View, tablers map[Code]pstable.Tabler, d *display.Display, updater TablerUpdater) *Manager {
	m := &Manager{
		view:    v,
		tablers: tablers,
		display: d,
		updater: updater,
	}
	// Initialize the current tabler to match the initial view
	m.updater.SetCurrentTabler(m.CurrentTabler())
	return m
}

// CurrentTabler returns the Tabler for the current view.
func (m *Manager) CurrentTabler() pstable.Tabler {
	return m.tablers[m.view.Get()]
}

// SetNext changes to the next view and updates the current tabler.
func (m *Manager) SetNext() {
	m.view.SetNext()
	m.updater.SetCurrentTabler(m.CurrentTabler())
}

// SetPrev changes to the previous view and updates the current tabler.
func (m *Manager) SetPrev() {
	m.view.SetPrev()
	m.updater.SetCurrentTabler(m.CurrentTabler())
}

// Set changes to the specified view code and updates the current tabler.
func (m *Manager) Set(code Code) {
	m.view.Set(code)
	m.updater.SetCurrentTabler(m.CurrentTabler())
}

// SetByName changes to the view with the given name and updates the current tabler.
// Returns an error if the name is not found or not selectable.
func (m *Manager) SetByName(name string) error {
	if err := m.view.SetByName(name); err != nil {
		return err
	}
	m.updater.SetCurrentTabler(m.CurrentTabler())
	return nil
}

// ToggleHelp toggles the help display flag.
func (m *Manager) ToggleHelp() {
	m.help = !m.help
}

// Help returns whether help is currently shown.
func (m *Manager) Help() bool {
	return m.help
}

// Display renders the current view (either help or the current tabler) to the screen.
func (m *Manager) Display() {
	if m.help {
		m.display.Display(display.Help)
	} else {
		m.display.Display(m.CurrentTabler())
	}
}

// ClearDisplay clears the screen.
func (m *Manager) ClearDisplay() {
	m.display.Clear()
}

// DisplayNext advances to the next view, clears the screen, and displays the new view.
func (m *Manager) DisplayNext() {
	m.SetNext()
	m.display.Clear()
	m.Display()
}

// DisplayPrev goes to the previous view, clears the screen, and displays the new view.
func (m *Manager) DisplayPrev() {
	m.SetPrev()
	m.display.Clear()
	m.Display()
}
