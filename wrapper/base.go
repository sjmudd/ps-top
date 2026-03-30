package wrapper

import (
	"fmt"
	"time"

	"github.com/sjmudd/ps-top/model"
)

// BaseWrapper[T, M] implements Tabler for any model M that satisfies
// model.Model[T]. It delegates to a content function provided at construction.
type BaseWrapper[T any, M model.Model[T]] struct {
	model     M
	name      string
	sortFn    func([]T)         // optional sorting function; nil = no sort
	hasData   func(T) bool      // predicate for counting rows with data; nil = count all rows
	contentFn func(T, T) string // formats a single row
}

// NewBaseWrapper creates a new BaseWrapper with the given model and options.
func NewBaseWrapper[T any, M model.Model[T]](
	model M,
	name string,
	sortFn func([]T),
	hasData func(T) bool,
	contentFn func(T, T) string,
) *BaseWrapper[T, M] {
	return &BaseWrapper[T, M]{
		model:     model,
		name:      name,
		sortFn:    sortFn,
		hasData:   hasData,
		contentFn: contentFn,
	}
}

// Collect implements Tabler.
func (bw *BaseWrapper[T, M]) Collect() {
	bw.model.Collect()
	if bw.sortFn != nil {
		results := bw.model.GetResults()
		bw.sortFn(results)
	}
}

// ResetStatistics implements Tabler.
func (bw *BaseWrapper[T, M]) ResetStatistics() {
	bw.model.ResetStatistics()
}

// HaveRelativeStats implements Tabler.
func (bw *BaseWrapper[T, M]) HaveRelativeStats() bool {
	return bw.model.HaveRelativeStats()
}

// FirstCollectTime implements Tabler.
func (bw *BaseWrapper[T, M]) FirstCollectTime() time.Time {
	return bw.model.GetFirstCollected()
}

// LastCollectTime implements Tabler.
func (bw *BaseWrapper[T, M]) LastCollectTime() time.Time {
	return bw.model.GetLastCollected()
}

// WantRelativeStats implements Tabler.
func (bw *BaseWrapper[T, M]) WantRelativeStats() bool {
	return bw.model.WantRelativeStats()
}

// RowContent implements Tabler.
func (bw *BaseWrapper[T, M]) RowContent() []string {
	results := bw.model.GetResults()
	n := len(results)
	return RowsFromGetter(n, func(i int) string {
		return bw.contentFn(results[i], bw.model.GetTotals())
	})
}

// TotalRowContent implements Tabler.
func (bw *BaseWrapper[T, M]) TotalRowContent() string {
	totals := bw.model.GetTotals()
	return TotalRowContent(totals, bw.contentFn)
}

// EmptyRowContent implements Tabler.
func (bw *BaseWrapper[T, M]) EmptyRowContent() string {
	return EmptyRowContent(bw.contentFn)
}

// Description implements Tabler.
func (bw *BaseWrapper[T, M]) Description() string {
	results := bw.model.GetResults()
	n := len(results)
	count := n
	if bw.hasData != nil {
		count = CountIf(n, func(i int) bool { return bw.hasData(results[i]) })
	}
	return fmt.Sprintf("%s %d rows", bw.name, count)
}

// GetModel returns the embedded model. Used by special wrappers like tableioops.
func (bw *BaseWrapper[T, M]) GetModel() M {
	return bw.model
}
