package model

import (
	"github.com/sjmudd/ps-top/global"
	"github.com/sjmudd/ps-top/model/filter"
)

// Config defines the minimal configuration required by data models.
type Config interface {
	WantRelativeStats() bool
	DatabaseFilter() *filter.DatabaseFilter
	Variables() *global.Variables
}
