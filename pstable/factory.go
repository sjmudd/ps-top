package pstable

import (
	"database/sql"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/wrapper/fileinfolatency"
	"github.com/sjmudd/ps-top/wrapper/memoryusage"
	"github.com/sjmudd/ps-top/wrapper/mutexlatency"
	"github.com/sjmudd/ps-top/wrapper/stageslatency"
	"github.com/sjmudd/ps-top/wrapper/tableiolatency"
	"github.com/sjmudd/ps-top/wrapper/tableioops"
	"github.com/sjmudd/ps-top/wrapper/tablelocklatency"
	"github.com/sjmudd/ps-top/wrapper/userlatency"
)

// TablerType defines the type of PS table data
type TablerType = int

const (
	FileIoLatency TablerType = iota
	LockLatency
	MemoryUsage
	MutexLatency
	StagesLatency
	TableIoLatency
	TableLockLatency
	UserLatency
)

// NewTable returns a Tabler of the defined type with the given parameters provided to the constructor
func NewTabler(tablerType TablerType, cfg *config.Config, db *sql.DB) Tabler {
	var t Tabler

	switch tablerType {
	case FileIoLatency:
		t = fileinfolatency.NewFileSummaryByInstance(cfg, db)
	case LockLatency:
		t = tablelocklatency.NewTableLockLatency(cfg, db)
	case MemoryUsage:
		t = memoryusage.NewMemoryUsage(cfg, db)
	case MutexLatency:
		t = mutexlatency.NewMutexLatency(cfg, db)
	case StagesLatency:
		t = stageslatency.NewStagesLatency(cfg, db)
	case TableIoLatency:
		t = tableiolatency.NewTableIoLatency(cfg, db)
	case UserLatency:
		t = userlatency.NewUserLatency(cfg, db)
	default:
		panic("NewTabler: invalid tablerType")
	}
	return t
}

// NewTableIoOps returns a tabler of this type given a shared backend existing Tabler configuration
func NewTableIoOps(latency Tabler) Tabler {
	if typedLatency, ok := latency.(*tableiolatency.Wrapper); ok {
		return tableioops.NewTableIoOps(typedLatency)
	}
	panic("NewTableIoOps: invalid type provided")
}
