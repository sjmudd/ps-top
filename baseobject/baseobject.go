// Package baseobject contains the library
// routines for base stuff of an object
package baseobject

import (
	"log"

	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/global"
	"github.com/sjmudd/ps-top/model/filter"
)

// BaseObject holds colllection times and a context
type BaseObject struct {
	CollectTime
	ctx *context.Context
}

// DatabaseFilter returns the context's DatabaseFilter()
func (o *BaseObject) DatabaseFilter() *filter.DatabaseFilter {
	return o.ctx.DatabaseFilter()
}

// SetContext sets the context in this object which can be used later.
// - it should always be defined (!= nil)
func (o *BaseObject) SetContext(ctx *context.Context) {
	if ctx == nil {
		log.Fatal("BaseObject.SetContext(ctx) ctx should not be nil")
	}
	o.ctx = ctx
}

// Variables returns a pointer to the global variables
func (o BaseObject) Variables() *global.Variables {
	if o.ctx == nil {
		log.Fatal("BaseObject.Variables() o.ctx should not be nil")
	}
	return o.ctx.Variables()
}

// WantRelativeStats indicates whether we want relative stats or not
// - FIXME and optmise me away
func (o BaseObject) WantRelativeStats() bool {
	if o.ctx == nil {
		log.Fatal("BaseObject.WantRelativeStats(): o.ctx should not be nil")
		return false
	}
	return o.ctx.WantRelativeStats()
}
