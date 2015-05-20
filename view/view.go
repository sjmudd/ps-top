package view

import (
	"log"

	"github.com/sjmudd/ps-top/lib"
)

// Type represents the type of information to view (as an int)
type Type int

// View* constants represent different views we can see
const (
	ViewLatency Type = iota // view the table latency information
	ViewOps     Type = iota // view the table information by number of operations
	ViewIO      Type = iota // view the file I/O information
	ViewLocks   Type = iota
	ViewUsers   Type = iota
	ViewMutex   Type = iota
	ViewStages  Type = iota
)

// View holds the integer type of view (maybe need to fix this setup)
type View struct {
	id Type
}

var (
	viewNames []string // maps View* to a string name
)

func init() {
	viewNames = []string{"table_io_latency", "table_io_ops", "file_io_latency", "table_lock_latency", "user_latency", "mutex_latency", "stages_latency"}
}

// SetNext changes the current view to the next one
func (s *View) SetNext() Type {
	if s.id < ViewStages {
		s.id++
	} else {
		s.id = ViewLatency
	}

	return s.Get()
}

// SetPrev changes the current view to the previous one
func (s *View) SetPrev() Type {
	if s.id > ViewLatency {
		s.id--
	} else {
		s.id = ViewStages
	}
	return s.Get()
}

// Set sets the view to the given view (by Type)
func (s *View) Set(viewType Type) {
	s.id = viewType
}

// SetByName sets the view based on its name.
// - If we provide an empty name then use the default.
// - If we don't provide a valid name then give an error
func (s *View) SetByName(name string) {
	if name == "" {
		lib.Logger.Println("View.SetByName(): name is empty so setting to:", ViewLatency.String())
		s.Set(ViewLatency)
		return
	}

	for i := range viewNames {
		if name == viewNames[i] {
			s.id = Type(i)
			lib.Logger.Println("View.SetByName(", name, ")")
			return
		}
	}

	// suggest what should be used
	allViews := ""
	for i := range viewNames {
		allViews = allViews + " " + viewNames[i]
	}

	// no need for now to trip off leading space from allViews.
	log.Fatal("Asked for a view name, '", name, "' which doesn't exist. Try one of:", allViews)
}

// Get returns the Type version of the current view
func (s View) Get() Type {
	return s.id
}

// GetName returns the string version of the current view
func (s View) GetName() string {
	return s.id.String()
}

func (s Type) String() string {
	return viewNames[s]
}
