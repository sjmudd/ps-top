package model

import (
	"time"

	"github.com/sjmudd/ps-top/config"
)

// BaseCollector encapsulates the common collection state and logic for all models.
// T is the row type (e.g., tableio.Row, memoryusage.Row)
// R is a slice of T (e.g., []Row, or a named type like Rows)
type BaseCollector[T any, R ~[]T] struct {
	config *config.Config
	db     QueryExecutor

	FirstCollected time.Time
	LastCollected  time.Time
	First          R // baseline snapshot
	Last           R // most recent raw data collection
	Results        R // processed results (after subtraction, etc.)
	Totals         T // totals row computed from results
	process        ProcessFunc[T, R]
}

// NewBaseCollector creates a new BaseCollector with the given config, database, and process function.
func NewBaseCollector[T any, R ~[]T](cfg *config.Config, db QueryExecutor, process ProcessFunc[T, R]) *BaseCollector[T, R] {
	return &BaseCollector[T, R]{
		config:  cfg,
		db:      db,
		process: process,
	}
}

// ProcessFunc defines the transformation from raw data to displayable results.
// It receives the last collected data and the baseline (first) data,
// and returns the processed results along with their totals.
type ProcessFunc[T any, R ~[]T] func(last, first R) (results R, totals T)

// WantRefreshFunc determines whether the baseline should be refreshed
// (i.e., copy last to first) based on current state.
type WantRefreshFunc = func() bool

// FetchFunc retrieves raw data from the database.
type FetchFunc[R any] func() (R, error)

// Collect orchestrates a full collection cycle:
// 1. Fetch raw data via fetchFunc
// 2. Optionally refresh baseline if wantRefresh returns true
// 3. Process data via the stored process function to produce results and totals
func (bc *BaseCollector[T, R]) Collect(
	fetch FetchFunc[R],
	wantRefresh WantRefreshFunc,
) {
	// Fetch the latest data
	last, err := fetch()
	if err != nil {
		// TODO: log error? For now, skip this collection cycle.
		return
	}

	// Update last snapshot and timestamp
	bc.Last = last
	bc.LastCollected = time.Now()
	if bc.FirstCollected.IsZero() {
		bc.FirstCollected = bc.LastCollected
	}

	// Refresh baseline if needed (e.g., on first collection or wrap-around)
	if wantRefresh() {
		bc.First = make(R, len(last))
		copy(bc.First, last)
		bc.FirstCollected = bc.LastCollected
	}

	// Process results and compute totals using the stored process function
	bc.Results, bc.Totals = bc.process(bc.Last, bc.First)
}

// ResetStatistics sets the baseline to the last collected values.
// This is used when the user requests a manual reset.
func (bc *BaseCollector[T, R]) ResetStatistics() {
	bc.First = make(R, len(bc.Last))
	copy(bc.First, bc.Last)
	bc.FirstCollected = bc.LastCollected
	bc.Results, bc.Totals = bc.process(bc.Last, bc.First)
}

// Config returns the collector's configuration
func (bc *BaseCollector[T, R]) Config() *config.Config {
	return bc.config
}

// DB returns the QueryExecutor (for use in fetch functions)
func (bc *BaseCollector[T, R]) DB() QueryExecutor {
	return bc.db
}

// Model is the interface that a data model must satisfy to be used with
// wrapper.BaseWrapper. It includes collection control, statistics queries,
// and data accessors.
type Model[T any] interface {
	Collect()
	ResetStatistics()
	HaveRelativeStats() bool
	WantRelativeStats() bool
	GetFirstCollected() time.Time
	GetLastCollected() time.Time
	GetResults() []T
	GetTotals() T
}

// GetResults returns the results slice as a []T.
// This provides interface-accessible access to the Results field.
func (bc *BaseCollector[T, R]) GetResults() []T {
	return []T(bc.Results)
}

// GetTotals returns the totals row.
func (bc *BaseCollector[T, R]) GetTotals() T {
	return bc.Totals
}

// GetFirstCollected returns the time of the first collection.
func (bc *BaseCollector[T, R]) GetFirstCollected() time.Time {
	return bc.FirstCollected
}

// GetLastCollected returns the time of the last collection.
func (bc *BaseCollector[T, R]) GetLastCollected() time.Time {
	return bc.LastCollected
}
