package app

import (
	"database/sql"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/tableio"
	"github.com/sjmudd/ps-top/pstable"
	"github.com/sjmudd/ps-top/view"
	"github.com/sjmudd/ps-top/wrapper/tableiolatency"
	"github.com/sjmudd/ps-top/wrapper/tableioops"
)

// DBCollector owns all the Tabler instances and coordinates data collection.
// It knows nothing about display, signals, or event loops - only database collection.
type DBCollector struct {
	config           *config.Config
	db               *sql.DB
	fileInfoLatency  pstable.Tabler
	tableIoLatency   pstable.Tabler
	tableIoOps       pstable.Tabler
	tableLockLatency pstable.Tabler
	mutexLatency     pstable.Tabler
	stagesLatency    pstable.Tabler
	memoryUsage      pstable.Tabler
	userLatency      pstable.Tabler
	currentTabler    pstable.Tabler
	currentView      view.View
}

// NewDBCollector creates and initializes all tablers.
func NewDBCollector(cfg *config.Config, db *sql.DB) *DBCollector {
	dc := &DBCollector{
		config: cfg,
		db:     db,
	}

	// Initialize all tablers
	dc.fileInfoLatency = pstable.NewTabler(pstable.FileIoLatency, cfg, db)
	// Create shared TableIo model for both latency and ops views
	sharedTableIo := tableio.NewTableIo(cfg, db)
	dc.tableIoLatency = tableiolatency.NewTableIoLatency(sharedTableIo)
	dc.tableIoOps = tableioops.NewTableIoOps(sharedTableIo)
	dc.tableLockLatency = pstable.NewTabler(pstable.TableLockLatency, cfg, db)
	dc.mutexLatency = pstable.NewTabler(pstable.MutexLatency, cfg, db)
	dc.stagesLatency = pstable.NewTabler(pstable.StagesLatency, cfg, db)
	dc.memoryUsage = pstable.NewTabler(pstable.MemoryUsage, cfg, db)
	dc.userLatency = pstable.NewTabler(pstable.UserLatency, cfg, db)

	return dc
}

// UpdateCurrentTabler sets currentTabler based on currentView.
// Extracted from App.UpdateCurrentTabler.
func (dc *DBCollector) UpdateCurrentTabler() {
	switch dc.currentView.Get() {
	case view.ViewLatency:
		dc.currentTabler = dc.tableIoLatency
	case view.ViewOps:
		dc.currentTabler = dc.tableIoOps
	case view.ViewIO:
		dc.currentTabler = dc.fileInfoLatency
	case view.ViewLocks:
		dc.currentTabler = dc.tableLockLatency
	case view.ViewUsers:
		dc.currentTabler = dc.userLatency
	case view.ViewMutex:
		dc.currentTabler = dc.mutexLatency
	case view.ViewStages:
		dc.currentTabler = dc.stagesLatency
	case view.ViewMemory:
		dc.currentTabler = dc.memoryUsage
	}
}

// Collect collects data for the current tabler.
// Extracted from App.Collect.
func (dc *DBCollector) Collect() {
	dc.currentTabler.Collect()
}

// CollectAll collects data for all tablers.
// Extracted from App.collectAll.
func (dc *DBCollector) CollectAll() {
	dc.fileInfoLatency.Collect()
	dc.tableLockLatency.Collect()
	dc.tableIoLatency.Collect()
	dc.userLatency.Collect()
	dc.stagesLatency.Collect()
	dc.mutexLatency.Collect()
	dc.memoryUsage.Collect()
}

// ResetAll resets statistics on all tablers.
// Extracted from App.resetStatistics.
func (dc *DBCollector) ResetAll() {
	dc.fileInfoLatency.ResetStatistics()
	dc.tableLockLatency.ResetStatistics()
	dc.tableIoLatency.ResetStatistics()
	dc.userLatency.ResetStatistics()
	dc.stagesLatency.ResetStatistics()
	dc.mutexLatency.ResetStatistics()
	dc.memoryUsage.ResetStatistics()
}

// SetView updates the current view and refreshes currentTabler.
func (dc *DBCollector) SetView(v view.View) {
	dc.currentView = v
	dc.UpdateCurrentTabler()
}

// CurrentTabler returns the currently selected tabler (for display).
func (dc *DBCollector) CurrentTabler() pstable.Tabler {
	return dc.currentTabler
}

// CurrentView returns the current view code.
func (dc *DBCollector) CurrentView() view.View {
	return dc.currentView
}

// SetViewByName sets the view by name (convenience wrapper).
func (dc *DBCollector) SetViewByName(name string) error {
	if err := dc.currentView.SetByName(name); err != nil {
		return err
	}
	dc.UpdateCurrentTabler()
	return nil
}
