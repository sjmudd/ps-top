package presenter

import (
	"fmt"
	"time"

	"github.com/sjmudd/ps-top/model"
)

// BasePresenter[T, M] implements Tabler for any model M that satisfies
// model.Model[T]. It delegates to a content function provided at construction.
type BasePresenter[T any, M model.Model[T]] struct {
	model     M
	name      string
	sortFn    func([]T)         // optional sorting function; nil = no sort
	hasData   func(T) bool      // predicate for counting rows with data; nil = count all rows
	contentFn func(T, T) string // formats a single row
}

// NewBasePresenter creates a new BasePresenter with the given model and options.
func NewBasePresenter[T any, M model.Model[T]](
	model M,
	name string,
	sortFn func([]T),
	hasData func(T) bool,
	contentFn func(T, T) string,
) *BasePresenter[T, M] {
	return &BasePresenter[T, M]{
		model:     model,
		name:      name,
		sortFn:    sortFn,
		hasData:   hasData,
		contentFn: contentFn,
	}
}

// Collect implements Tabler.
func (bp *BasePresenter[T, M]) Collect() {
	bp.model.Collect()
	if bp.sortFn != nil {
		results := bp.model.GetResults()
		bp.sortFn(results)
	}
}

// ResetStatistics implements Tabler.
func (bp *BasePresenter[T, M]) ResetStatistics() {
	bp.model.ResetStatistics()
}

// HaveRelativeStats implements Tabler.
func (bp *BasePresenter[T, M]) HaveRelativeStats() bool {
	return bp.model.HaveRelativeStats()
}

// FirstCollectTime implements Tabler.
func (bp *BasePresenter[T, M]) FirstCollectTime() time.Time {
	return bp.model.GetFirstCollected()
}

// LastCollectTime implements Tabler.
func (bp *BasePresenter[T, M]) LastCollectTime() time.Time {
	return bp.model.GetLastCollected()
}

// WantRelativeStats implements Tabler.
func (bp *BasePresenter[T, M]) WantRelativeStats() bool {
	return bp.model.WantRelativeStats()
}

// RowContent implements Tabler.
func (bp *BasePresenter[T, M]) RowContent() []string {
	results := bp.model.GetResults()
	n := len(results)
	return RowsFromGetter(n, func(i int) string {
		return bp.contentFn(results[i], bp.model.GetTotals())
	})
}

// TotalRowContent implements Tabler.
func (bp *BasePresenter[T, M]) TotalRowContent() string {
	totals := bp.model.GetTotals()
	return TotalRowContent(totals, bp.contentFn)
}

// EmptyRowContent implements Tabler.
func (bp *BasePresenter[T, M]) EmptyRowContent() string {
	return EmptyRowContent(bp.contentFn)
}

// Description implements Tabler.
func (bp *BasePresenter[T, M]) Description() string {
	results := bp.model.GetResults()
	n := len(results)
	count := n
	if bp.hasData != nil {
		count = CountIf(n, func(i int) bool { return bp.hasData(results[i]) })
	}
	return fmt.Sprintf("%s %d rows", bp.name, count)
}

// GetModel returns the embedded model. Used by special presenters like tableioops.
func (bp *BasePresenter[T, M]) GetModel() M {
	return bp.model
}
