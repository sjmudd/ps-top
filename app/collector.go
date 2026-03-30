package app

import (
	"database/sql"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/tableio"
	"github.com/sjmudd/ps-top/presenter/tableiolatency"
	"github.com/sjmudd/ps-top/presenter/tableioops"
	"github.com/sjmudd/ps-top/pstable"
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

// CurrentTabler returns the currently selected tabler (for display).
func (dc *DBCollector) CurrentTabler() pstable.Tabler {
	return dc.currentTabler
}

// SetCurrentTabler updates the currently selected tabler.
// Used by ViewManager when the view changes.
func (dc *DBCollector) SetCurrentTabler(t pstable.Tabler) {
	dc.currentTabler = t
}
