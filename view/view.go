package view

import (
	"database/sql"
	"errors"
	"log"

	"github.com/sjmudd/ps-top/logger"
	"github.com/sjmudd/ps-top/table"
)

// Code represents the type of information to view (as an int)
type Code int

// View* constants represent different views we can see
const (
	ViewNone    Code = iota // view nothing (should never be set)
	ViewLatency Code = iota // view the table latency information
	ViewOps     Code = iota // view the table information by number of operations
	ViewIO      Code = iota // view the file I/O information
	ViewLocks   Code = iota // view lock information
	ViewUsers   Code = iota // view user information
	ViewMutex   Code = iota // view mutex information
	ViewStages  Code = iota // view SQL stages information
	ViewMemory  Code = iota // view memory usage (5.7 only)
)

// View holds the integer type of view (maybe need to fix this setup)
type View struct {
	code Code
}

var (
	names  map[Code]string       // map View* to a string name
	tables map[Code]table.Access // map a view to a table name and whether it's selectable or not

	nextView map[Code]Code // map from one view to the next taking into account invalid views
	prevView map[Code]Code // map from one view to the next taking into account invalid views
)

func init() {
	names = map[Code]string{
		ViewLatency: "table_io_latency",
		ViewOps:     "table_io_ops",
		ViewIO:      "file_io_latency",
		ViewLocks:   "table_lock_latency",
		ViewUsers:   "user_latency",
		ViewMutex:   "mutex_latency",
		ViewStages:  "stages_latency",
		ViewMemory:  "memory_usage",
	}

	tables = map[Code]table.Access{
		ViewLatency: table.NewAccess("performance_schema", "table_io_waits_summary_by_table"),
		ViewOps:     table.NewAccess("performance_schema", "table_io_waits_summary_by_table"),
		ViewIO:      table.NewAccess("performance_schema", "file_summary_by_instance"),
		ViewLocks:   table.NewAccess("performance_schema", "table_lock_waits_summary_by_table"),
		ViewUsers:   table.NewAccess("information_schema", "processlist"),
		ViewMutex:   table.NewAccess("performance_schema", "events_waits_summary_global_by_event_name"),
		ViewStages:  table.NewAccess("performance_schema", "events_stages_summary_global_by_event_name"),
		ViewMemory:  table.NewAccess("performance_schema", "memory_summary_global_by_event_name"),
	}
}

// ValidateViews check which views are readable. If none are we give a fatal error
func ValidateViews(dbh *sql.DB) error {
	var count int
	var status string
	logger.Println("Validating access to views...")

	// determine which of the defined views is valid because the underlying table access works
	for v := range names {
		ta := tables[v]
		e := ta.CheckSelectError(dbh)
		suffix := ""
		if e == nil {
			status = "is"
			count++
		} else {
			status = "IS NOT"
			suffix = " " + e.Error()
		}
		tables[v] = ta
		logger.Println(v.String() + ": " + ta.Name() + " " + status + " SELECTable" + suffix)
	}

	if count == 0 {
		return errors.New("None of the required tables are SELECTable. Giving up")
	}
	logger.Println(count, "of", len(names), "view(s) are SELECTable, continuing")

	setPrevAndNextViews()

	return nil
}

/* set the previous and next views taking into account any invalid views

name     selectable?    prev      next
----     -----------    ----      ----
v1       false          v4        v2
v2       true           v4        v4
v3       false          v2        v4
v4       true           v2        v2
v5       false          v4        v2

*/

func setPrevAndNextViews() {
	logger.Println("view.setPrevAndNextViews()...")
	nextView = make(map[Code]Code)
	prevView = make(map[Code]Code)

	// reset values
	for v := range names {
		nextView[v] = ViewNone
		prevView[v] = ViewNone
	}

	// Cleaner way to do this? Probably. Fix later.
	prevCodeOrder := []Code{ViewMemory, ViewStages, ViewMutex, ViewUsers, ViewLocks, ViewIO, ViewOps, ViewLatency}
	nextCodeOrder := []Code{ViewLatency, ViewOps, ViewIO, ViewLocks, ViewUsers, ViewMutex, ViewStages, ViewMemory}
	prevView = setValidByValues(prevCodeOrder)
	nextView = setValidByValues(nextCodeOrder)

	// print out the results
	logger.Println("Final mapping of view order:")
	for i := range nextCodeOrder {
		logger.Println("view:", nextCodeOrder[i], ", prev:", prevView[nextCodeOrder[i]], ", next:", nextView[nextCodeOrder[i]])
	}
}

// setValidNextByValues returns a map of Code -> Code where the mapping points to the "next"
// Code. The order is determined by the input Code slice. Only Selectable Views are considered
// for the mapping with the other views pointing to the first Code provided.
func setValidByValues(orderedCodes []Code) map[Code]Code {
	logger.Println("view.setValidByValues()")
	orderedMap := make(map[Code]Code)

	// reset orderedCodes
	for i := range orderedCodes {
		orderedMap[orderedCodes[i]] = ViewNone
	}

	first, last := ViewNone, ViewNone

	// first pass, try to find values and point forward to next position if known.
	// we must find at least one value view in the first pass.
	for i := range []int{1, 2} {
		for i := range orderedCodes {
			currentPos := orderedCodes[i]
			if tables[currentPos].SelectError() == nil {
				if first == ViewNone {
					first = currentPos
				}
				if last != ViewNone {
					orderedMap[last] = currentPos
				}
				last = currentPos
			}
		}
		if i == 1 {
			// not found a valid view so something is up. Give up!
			if first == ViewNone {
				log.Panic("setValidByValues() can't find a Selectable view! (shouldn't be here)")
			}
		}
	}

	// final pass viewNone entries should point to first
	for i := range orderedCodes {
		currentPos := orderedCodes[i]
		if tables[currentPos].SelectError() != nil {
			orderedMap[currentPos] = first
		}
	}

	return orderedMap
}

// SetNext changes the current view to the next one
func (v *View) SetNext() Code {
	v.code = nextView[v.code]

	return v.code
}

// SetPrev changes the current view to the previous one
func (v *View) SetPrev() Code {
	v.code = prevView[v.code]

	return v.code
}

// Set sets the view to the given view (by Code)
func (v *View) Set(viewCode Code) {
	v.code = viewCode

	if tables[v.code].SelectError() != nil {
		v.code = nextView[v.code]
	}
}

// SetByName sets the view based on its name.
// - If we provide an empty name then use the default.
// - If we don't provide a valid name then give an error
func (v *View) SetByName(name string) {
	logger.Println("View.SetByName(" + name + ")")
	if name == "" {
		logger.Println("View.SetByName(): name is empty so setting to:", ViewLatency.String())
		v.Set(ViewLatency)
		return
	}

	for i := range names {
		if name == names[i] {
			v.code = Code(i)
			logger.Println("View.SetByName(", name, ")")
			return
		}
	}

	// suggest what should be used
	allViews := ""
	for i := range names {
		allViews = allViews + " " + names[i]
	}

	// no need for now to strip off leading space from allViews.
	log.Fatal("Asked for a view name, '", name, "' which doesn't exist. Try one of:", allViews)
}

// Get returns the Code version of the current view
func (v View) Get() Code {
	return v.code
}

// Name returns the string version of the current view
func (v View) Name() string {
	return v.code.String()
}

func (s Code) String() string {
	return names[s]
}
