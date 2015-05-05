package view

import (
	"log"

	"github.com/sjmudd/pstop/lib"
)

// what information to view
type ViewType int

const (
	ViewLatency ViewType = iota
	ViewOps     ViewType = iota
	ViewIO      ViewType = iota
	ViewLocks   ViewType = iota
	ViewUsers   ViewType = iota
	ViewMutex   ViewType = iota
	ViewStages  ViewType = iota
)

type View struct {
	id ViewType
}

var view_names []string

func init() {
	view_names = []string{"table_io_latency", "table_io_ops", "file_io_latency", "table_lock_latency", "user_latency", "mutex_latency", "stages_latency"}
}

// set the next view
func (s *View) SetNext() ViewType {
	if s.id <= ViewStages {
		s.id++
	} else {
		s.id = ViewLatency
	}

	return s.Get()
}

// set the previous view
func (s *View) SetPrev() ViewType {
	if s.id > ViewLatency {
		s.id--
	} else {
		s.id = ViewStages
	}
	return s.Get()
}

// set the view by id
func (s *View) Set(view_type ViewType) {
	s.id = view_type
}

// set the view based on its name.
// - If we provide an empty name then use the default.
// - If we don't provide a valid name then give an error
func (s *View) SetByName(name string) {
	if name == "" {
		lib.Logger.Println("View.SetByName(): name is empty so setting to:", ViewLatency.String())
		s.Set(ViewLatency)
		return
	}

	for i := range view_names {
		if name == view_names[i] {
			s.id = ViewType(i)
			lib.Logger.Println("View.SetByName(", name, ")")
			return
		}
	}

	// suggest what should be used
	all_views := ""
	for i := range view_names {
		all_views = all_views + " " + view_names[i]
	}

	// no need for now to trip off leading space from all_views.
	log.Fatal("Asked for a view name, '", name, "' which doesn't exist. Try one of:", all_views)
}

// return the current view
func (s View) Get() ViewType {
	return s.id
}

// get the name of the current view
func (s View) GetName() string {
	return view_names[s.id]
}

func (s ViewType) String() string {
	return view_names[s]
}
