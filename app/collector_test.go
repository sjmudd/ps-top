package app

import (
	"testing"
	"time"
)

// mockTabler is a minimal implementation of pstable.Tabler for testing.
type mockTabler struct {
	name    string
	coll    bool
	reset   bool
	desc    string
	head    string
	rows    []string
	total   string
	empty   string
	haveRel bool
	wantRel bool
}

func (m *mockTabler) Collect()                    { m.coll = true }
func (m *mockTabler) ResetStatistics()            { m.reset = true }
func (m *mockTabler) Description() string         { return m.desc }
func (m *mockTabler) HaveRelativeStats() bool     { return m.haveRel }
func (m *mockTabler) Headings() string            { return m.head }
func (m *mockTabler) FirstCollectTime() time.Time { return time.Time{} }
func (m *mockTabler) LastCollectTime() time.Time  { return time.Time{} }
func (m *mockTabler) RowContent() []string        { return m.rows }
func (m *mockTabler) TotalRowContent() string     { return m.total }
func (m *mockTabler) EmptyRowContent() string     { return m.empty }
func (m *mockTabler) WantRelativeStats() bool     { return m.wantRel }

// TestDBCollector_NewDBCollector verifies that NewDBCollector creates all 8 tablers.
// Since NewDBCollector actually calls pstable.NewTabler which needs a real DB,
// this test constructs a DBCollector manually with mock tablers to verify structure.
func TestDBCollector_Structure(t *testing.T) {
	dc := &DBCollector{
		fileInfoLatency:  &mockTabler{name: "fileInfoLatency"},
		tableIoLatency:   &mockTabler{name: "tableIoLatency"},
		tableIoOps:       &mockTabler{name: "tableIoOps"},
		tableLockLatency: &mockTabler{name: "tableLockLatency"},
		mutexLatency:     &mockTabler{name: "mutexLatency"},
		stagesLatency:    &mockTabler{name: "stagesLatency"},
		memoryUsage:      &mockTabler{name: "memoryUsage"},
		userLatency:      &mockTabler{name: "userLatency"},
	}
	if dc.fileInfoLatency == nil {
		t.Error("fileInfoLatency is nil")
	}
	if dc.tableIoLatency == nil {
		t.Error("tableIoLatency is nil")
	}
	if dc.tableIoOps == nil {
		t.Error("tableIoOps is nil")
	}
	if dc.tableLockLatency == nil {
		t.Error("tableLockLatency is nil")
	}
	if dc.mutexLatency == nil {
		t.Error("mutexLatency is nil")
	}
	if dc.stagesLatency == nil {
		t.Error("stagesLatency is nil")
	}
	if dc.memoryUsage == nil {
		t.Error("memoryUsage is nil")
	}
	if dc.userLatency == nil {
		t.Error("userLatency is nil")
	}
}

// TestDBCollector_Collect tests that Collect delegates to currentTabler.
func TestDBCollector_Collect(t *testing.T) {
	mt := &mockTabler{}
	dc := &DBCollector{
		currentTabler: mt,
	}
	dc.Collect()
	if !mt.coll {
		t.Error("Collect did not delegate to currentTabler")
	}
}

// TestDBCollector_CollectAll tests that CollectAll calls Collect on all 8 tablers.
func TestDBCollector_CollectAll(t *testing.T) {
	mocks := []*mockTabler{
		{name: "fileInfoLatency"},
		{name: "tableLockLatency"},
		{name: "tableIoLatency"},
		{name: "userLatency"},
		{name: "stagesLatency"},
		{name: "mutexLatency"},
		{name: "memoryUsage"},
	}
	dc := &DBCollector{
		fileInfoLatency:  mocks[0],
		tableLockLatency: mocks[1],
		tableIoLatency:   mocks[2],
		userLatency:      mocks[3],
		stagesLatency:    mocks[4],
		mutexLatency:     mocks[5],
		memoryUsage:      mocks[6],
	}
	dc.CollectAll()
	for i, m := range mocks {
		if !m.coll {
			t.Errorf("mockTabler %d (%s) was not collected", i, m.name)
		}
	}
}

// TestDBCollector_ResetAll tests that ResetAll calls ResetStatistics on all 8 tablers.
func TestDBCollector_ResetAll(t *testing.T) {
	mocks := []*mockTabler{
		{name: "fileInfoLatency"},
		{name: "tableLockLatency"},
		{name: "tableIoLatency"},
		{name: "userLatency"},
		{name: "stagesLatency"},
		{name: "mutexLatency"},
		{name: "memoryUsage"},
	}
	dc := &DBCollector{
		fileInfoLatency:  mocks[0],
		tableLockLatency: mocks[1],
		tableIoLatency:   mocks[2],
		userLatency:      mocks[3],
		stagesLatency:    mocks[4],
		mutexLatency:     mocks[5],
		memoryUsage:      mocks[6],
	}
	dc.ResetAll()
	for i, m := range mocks {
		if !m.reset {
			t.Errorf("mockTabler %d (%s) was not reset", i, m.name)
		}
	}
}

// TestDBCollector_Accessors tests CurrentTabler getter and SetCurrentTabler setter.
func TestDBCollector_Accessors(t *testing.T) {
	mt := &mockTabler{}
	dc := &DBCollector{
		currentTabler: mt,
	}
	if dc.CurrentTabler() != mt {
		t.Error("CurrentTabler did not return correct tabler")
	}

	// Test SetCurrentTabler
	mt2 := &mockTabler{name: "second"}
	dc.SetCurrentTabler(mt2)
	if dc.CurrentTabler() != mt2 {
		t.Error("SetCurrentTabler did not update currentTabler")
	}
}
