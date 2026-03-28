package view

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/sjmudd/ps-top/log"
	"github.com/sjmudd/ps-top/model/processlist"
)

// Code represents the type of information to view (as an int)
type Code int

// View* constants represent different views we can see
const (
	ViewNone    Code = iota // view nothing (should never be set)
	ViewLatency             // view the table latency information
	ViewOps                 // view the table information by number of operations
	ViewIO                  // view the file I/O information
	ViewLocks               // view lock information
	ViewUsers               // view user information
	ViewMutex               // view mutex information
	ViewStages              // view SQL stages information
	ViewMemory              // view memory usage (5.7+)
)

// viewDef holds the static and dynamic definition of a view
type viewDef struct {
	code       Code
	name       string // view name for display/URL
	table      string // fully qualified table name (database.table)
	selectable bool   // whether the table is accessible (determined at runtime)
}

// View holds the current view state and a reference to the manager
type View struct {
	manager *viewManager
	code    Code
}

// viewManager manages all views and their ordering
type viewManager struct {
	views       []viewDef    // all selectable views in display order
	codeToIndex map[Code]int // quick lookup from code to index
}

// allViewsDef contains the static definition for every view code
// The order in this slice defines the default display order
var allViewsDef = []viewDef{
	{ViewLatency, "table_io_latency", "performance_schema.table_io_waits_summary_by_table", false},
	{ViewOps, "table_io_ops", "performance_schema.table_io_waits_summary_by_table", false},
	{ViewIO, "file_io_latency", "performance_schema.file_summary_by_instance", false},
	{ViewLocks, "table_lock_latency", "performance_schema.table_lock_waits_summary_by_table", false},
	{ViewUsers, "user_latency", "processlist", false}, // processlist table resolved later
	{ViewMutex, "mutex_latency", "performance_schema.events_waits_summary_global_by_event_name", false},
	{ViewStages, "stages_latency", "performance_schema.events_stages_summary_global_by_event_name", false},
	{ViewMemory, "memory_usage", "performance_schema.memory_summary_global_by_event_name", false},
}

// SetupAndValidate creates a new view manager, validates table access,
// and returns a View set to the requested name (or default if empty).
// It is the main entry point for initializing the view system.
func SetupAndValidate(name string, db *sql.DB) (View, error) {
	log.Printf("view.SetupAndValidate(%q, db)", name)

	// Resolve processlist table schema (depends on MySQL version)
	defs := make([]viewDef, len(allViewsDef))
	copy(defs, allViewsDef) // shallow copy so we can modify the table field
	for i := range defs {
		if defs[i].code == ViewUsers {
			havePS, err := processlist.HavePerformanceSchema(db)
			if err != nil {
				return View{}, fmt.Errorf("SetupAndValidate: %w", err)
			}
			if havePS {
				defs[i].table = "performance_schema.processlist"
			} else {
				defs[i].table = "information_schema.processlist"
			}
		}
	}

	// Check which views are actually accessible
	selectableViews := make([]viewDef, 0, len(defs))
	for i := range defs {
		def := &defs[i]
		if def.canSelect(db) {
			def.selectable = true
			selectableViews = append(selectableViews, *def)
		} else {
			log.Printf("View %s (%s) is NOT SELECTable", def.name, def.table)
		}
	}

	if len(selectableViews) == 0 {
		return View{}, fmt.Errorf("no views are SELECTable")
	}

	log.Printf("%d of %d views are SELECTable, continuing", len(selectableViews), len(defs))

	// Build the manager
	manager := &viewManager{
		views:       selectableViews,
		codeToIndex: make(map[Code]int, len(selectableViews)),
	}
	for i, def := range selectableViews {
		manager.codeToIndex[def.code] = i
	}

	// Create initial view
	v := View{
		manager: manager,
	}

	// Set the requested view name (or default)
	if err := v.SetByName(name); err != nil {
		return View{}, err
	}

	return v, nil
}

// canSelect tests if the view's table can be queried
func (def *viewDef) canSelect(db *sql.DB) bool {
	var dummy int
	err := db.QueryRow("SELECT 1 FROM " + def.table + " LIMIT 1").Scan(&dummy)
	return err == nil || err == sql.ErrNoRows
}

// SetNext changes to the next view (wraps around)
func (v *View) SetNext() Code {
	idx := v.manager.codeToIndex[v.code]
	idx = (idx + 1) % len(v.manager.views)
	v.code = v.manager.views[idx].code
	return v.code
}

// SetPrev changes to the previous view (wraps around)
func (v *View) SetPrev() Code {
	idx := v.manager.codeToIndex[v.code]
	idx = (idx - 1 + len(v.manager.views)) % len(v.manager.views)
	v.code = v.manager.views[idx].code
	return v.code
}

// Set sets the view to the specified code if it's selectable.
// If not selectable, falls back to the first selectable view.
func (v *View) Set(viewCode Code) {
	if _, ok := v.manager.codeToIndex[viewCode]; ok {
		v.code = viewCode
	} else {
		// fallback to first view
		v.code = v.manager.views[0].code
	}
}

// SetByName sets the view by its string name.
// Empty name selects the first view (ViewLatency if available).
// Returns error if the name is not found or not selectable.
func (v *View) SetByName(name string) error {
	log.Println("View.SetByName(" + name + ")")

	if name == "" {
		v.code = v.manager.views[0].code
		log.Println("View.SetByName(): empty name, set to:", v.String())
		return nil
	}

	// Look up the code from the static allViewsDef list
	for _, def := range allViewsDef {
		if def.name == name {
			if _, ok := v.manager.codeToIndex[def.code]; ok {
				v.code = def.code
				log.Println("View.SetByName(): set to", name)
				return nil
			}
		}
	}

	// not found or not selectable
	allViews := make([]string, 0, len(v.manager.views))
	for _, def := range v.manager.views {
		allViews = append(allViews, def.name)
	}
	return fmt.Errorf("view name '%s' not found or not selectable. Available: %s", name, strings.Join(allViews, ", "))
}

// Get returns the current view Code
func (v View) Get() Code {
	return v.code
}

// Name returns the string name of the current view
func (v View) Name() string {
	if idx, ok := v.manager.codeToIndex[v.code]; ok {
		return v.manager.views[idx].name
	}
	return ""
}

// String returns the string name (same as Name)
func (v View) String() string {
	return v.Name()
}
