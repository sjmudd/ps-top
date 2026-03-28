package view

import (
	"testing"
)

// mockViewManager creates a viewManager with the given view definitions.
// Only selectable views are included, mirroring the real SetupAndValidate.
func mockViewManager(defs []viewDef) *viewManager {
	manager := &viewManager{
		views:       make([]viewDef, 0, len(defs)),
		codeToIndex: make(map[Code]int, len(defs)),
	}
	for _, def := range defs {
		if def.selectable {
			manager.views = append(manager.views, def)
			manager.codeToIndex[def.code] = len(manager.views) - 1
		}
	}
	return manager
}

// mockView creates a View with the given manager and initial code.
// This is for testing only.
func mockView(manager *viewManager, initialCode Code) View {
	return View{
		manager: manager,
		code:    initialCode,
	}
}

// TestViewSetNext tests that SetNext cycles through views correctly.
func TestViewSetNext(t *testing.T) {
	defs := []viewDef{
		{code: ViewLatency, name: "table_io_latency", table: "performance_schema.table1", selectable: true},
		{code: ViewOps, name: "table_io_ops", table: "performance_schema.table2", selectable: true},
		{code: ViewIO, name: "file_io_latency", table: "performance_schema.table3", selectable: true},
	}
	manager := mockViewManager(defs)
	v := mockView(manager, ViewLatency)

	if v.SetNext() != ViewOps {
		t.Errorf("SetNext from ViewLatency expected ViewOps, got %v", v.Get())
	}
	if v.Get() != ViewOps {
		t.Errorf("after SetNext, view code should be ViewOps, got %v", v.Get())
	}

	if v.SetNext() != ViewIO {
		t.Errorf("SetNext from ViewOps expected ViewIO, got %v", v.Get())
	}

	// Should wrap around
	if v.SetNext() != ViewLatency {
		t.Errorf("SetNext should wrap around from ViewIO to ViewLatency, got %v", v.Get())
	}
}

// TestViewSetPrev tests that SetPrev cycles through views correctly.
func TestViewSetPrev(t *testing.T) {
	defs := []viewDef{
		{code: ViewLatency, name: "table_io_latency", table: "performance_schema.table1", selectable: true},
		{code: ViewOps, name: "table_io_ops", table: "performance_schema.table2", selectable: true},
		{code: ViewIO, name: "file_io_latency", table: "performance_schema.table3", selectable: true},
	}
	manager := mockViewManager(defs)
	v := mockView(manager, ViewOps)

	if v.SetPrev() != ViewLatency {
		t.Errorf("SetPrev from ViewOps expected ViewLatency, got %v", v.Get())
	}

	// Should wrap around
	if v.SetPrev() != ViewIO {
		t.Errorf("SetPrev should wrap around from ViewLatency to ViewIO, got %v", v.Get())
	}
}

// TestViewSet tests setting view code directly.
func TestViewSet(t *testing.T) {
	defs := []viewDef{
		{code: ViewLatency, name: "table_io_latency", table: "performance_schema.table1", selectable: true},
		{code: ViewOps, name: "table_io_ops", table: "performance_schema.table2", selectable: true},
		{code: ViewIO, name: "file_io_latency", table: "performance_schema.table3", selectable: false}, // not selectable
	}
	manager := mockViewManager(defs)

	// Valid, selectable code
	v := mockView(manager, ViewLatency)
	v.Set(ViewOps)
	if v.Get() != ViewOps {
		t.Errorf("Set(ViewOps) failed, got %v", v.Get())
	}

	// Non-selectable code should fall back to first selectable (ViewLatency)
	v.Set(ViewIO)
	if v.Get() != ViewLatency {
		t.Errorf("Set(non-selectable) should fall back to first selectable, got %v", v.Get())
	}
}

// TestViewName tests that Name() returns the correct view name.
func TestViewName(t *testing.T) {
	defs := []viewDef{
		{code: ViewLatency, name: "table_io_latency", table: "performance_schema.table1", selectable: true},
		{code: ViewOps, name: "table_io_ops", table: "performance_schema.table2", selectable: true},
	}
	manager := mockViewManager(defs)
	v := mockView(manager, ViewOps)

	if name := v.Name(); name != "table_io_ops" {
		t.Errorf("View.Name() expected 'table_io_ops', got %q", name)
	}
}

// TestViewNameUnknown tests that Name() returns empty string for unknown code.
func TestViewNameUnknown(t *testing.T) {
	defs := []viewDef{
		{code: ViewLatency, name: "table_io_latency", table: "performance_schema.table1", selectable: true},
	}
	manager := mockViewManager(defs)
	v := mockView(manager, ViewMutex) // not in manager

	if name := v.Name(); name != "" {
		t.Errorf("View.Name() for unknown code should return empty string, got %q", name)
	}
}

// TestSetByNameInvalid tests that SetByName returns error for unknown view name.
// This test directly uses SetByName and checks error handling.
func TestSetByNameInvalid(t *testing.T) {
	// The real SetByName depends on the global allViewsDef. For this test,
	// we directly check that providing a name that doesn't exist in allViewsDef
	// returns an error. Since we cannot easily mock allViewsDef, this test uses
	// the real global but with a name that definitely doesn't match.
	// However, we need a valid manager. We'll create one with a known selectable view.
	defs := []viewDef{
		{code: ViewLatency, name: "table_io_latency", table: "performance_schema.table1", selectable: true},
	}
	manager := mockViewManager(defs)
	v := mockView(manager, ViewLatency)

	// Use a nonsense name
	err := v.SetByName("this_name_does_not_exist")
	if err == nil {
		t.Fatal("SetByName(\"this_name_does_not_exist\") expected error, got nil")
	}
	// Should not change current view
	if v.Get() != ViewLatency {
		t.Errorf("SetByName error should not change view, got %v", v.Get())
	}
}

// TestSetByNameEmpty tests that SetByName with empty string selects first view.
func TestSetByNameEmpty(t *testing.T) {
	defs := []viewDef{
		{code: ViewLatency, name: "table_io_latency", table: "performance_schema.table1", selectable: true},
		{code: ViewOps, name: "table_io_ops", table: "performance_schema.table2", selectable: true},
	}
	manager := mockViewManager(defs)
	v := mockView(manager, ViewOps)

	err := v.SetByName("")
	if err != nil {
		t.Fatalf("SetByName(\"\") unexpected error: %v", err)
	}
	if v.Get() != ViewLatency {
		t.Errorf("SetByName(\"\") should select first view (ViewLatency), got %v", v.Get())
	}
}

// TestSetByNameValid tests that SetByName sets the correct view using a real view name.
// This test works with the global allViewsDef so we must use the exact names it defines.
func TestSetByNameValid(t *testing.T) {
	// Build a manager containing a view that matches one in the real allViewsDef.
	// We'll use ViewOps which has name "table_io_ops" in allViewsDef.
	defs := []viewDef{
		{code: ViewLatency, name: "table_io_latency", table: "performance_schema.table_io_waits_summary_by_table", selectable: true},
		{code: ViewOps, name: "table_io_ops", table: "performance_schema.table_io_waits_summary_by_table", selectable: true},
	}
	manager := mockViewManager(defs)
	v := mockView(manager, ViewLatency)

	// Use the real view name from allViewsDef
	err := v.SetByName("table_io_ops")
	if err != nil {
		t.Fatalf("SetByName(\"table_io_ops\") unexpected error: %v", err)
	}
	if v.Get() != ViewOps {
		t.Errorf("SetByName(\"table_io_ops\") expected ViewOps, got %v", v.Get())
	}
}
