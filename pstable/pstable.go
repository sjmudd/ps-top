// Package pstable contains the library routines for managing a
// generic performance_schema table via an interface definition.
package pstable

import (
	"database/sql"
	"time"

	"github.com/sjmudd/ps-top/log"
	"github.com/sjmudd/ps-top/model"
	"github.com/sjmudd/ps-top/model/tableio"
	"github.com/sjmudd/ps-top/presenter/fileinfolatency"
	"github.com/sjmudd/ps-top/presenter/memoryusage"
	"github.com/sjmudd/ps-top/presenter/mutexlatency"
	"github.com/sjmudd/ps-top/presenter/stageslatency"
	"github.com/sjmudd/ps-top/presenter/tableiolatency"
	"github.com/sjmudd/ps-top/presenter/tablelocklatency"
	"github.com/sjmudd/ps-top/presenter/userlatency"
)

// TablerType defines the type of PS table data
type TablerType = int

const (
	FileIoLatency TablerType = iota
	MemoryUsage
	MutexLatency
	StagesLatency
	TableIoLatency
	TableLockLatency
	UserLatency
)

// Tabler is the interface for access to performance_schema rows
type Tabler interface {
	Collect()                    // Collect collects data for the table from the database
	Description() string         // return a description of the metric
	EmptyRowContent() string     // return an empty row formatted as a string
	HaveRelativeStats() bool     // do we have relative stats in the provided data?
	Headings() string            // heading for text output
	FirstCollectTime() time.Time // time of first collection
	LastCollectTime() time.Time  // time of last collection
	RowContent() []string        // a list of text formatted row data
	ResetStatistics()            // resets the statistics for this data
	TotalRowContent() string     // text formatted data for the "total" footer
	WantRelativeStats() bool     // do we want relative stats?
}

// NewTabler returns a Tabler of the requested tablerType and parameters
func NewTabler(tablerType TablerType, cfg model.Config, db *sql.DB) Tabler {
	var t Tabler

	log.Printf("NewTabler(%v,%v,%v)\n", tablerType, cfg, db)

	switch tablerType {
	case FileIoLatency:
		t = fileinfolatency.NewFileSummaryByInstance(cfg, db)
	case TableLockLatency:
		t = tablelocklatency.NewTableLockLatency(cfg, db)
	case MemoryUsage:
		t = memoryusage.NewMemoryUsage(cfg, db)
	case MutexLatency:
		t = mutexlatency.NewMutexLatency(cfg, db)
	case StagesLatency:
		t = stageslatency.NewStagesLatency(cfg, db)
	case TableIoLatency:
		// Create a dedicated TableIo model for this latency presenter.
		// If both latency and ops views are needed, create a shared model and pass
		// to both tableiolatency.NewTableIoLatency and tableioops.NewTableIoOps directly.
		model := tableio.NewTableIo(cfg, db)
		t = tableiolatency.NewTableIoLatency(model)
	case UserLatency:
		t = userlatency.NewUserLatency(cfg, db)
	default:
		log.Printf("NewTabler: invalid tableType: %v", tablerType)
		panic("NewTabler: invalid tablerType")
	}

	log.Printf("NewTabler: t initialised to %v+\n", t)

	return t
}
