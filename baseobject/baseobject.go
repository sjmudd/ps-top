// Package baseobject contains the library
// routines for base stuff of an object
package baseobject

import (
	"log"
	"time"

	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/global"
)

type CollectTime struct {
	first time.Time // the first collection time (for relative data)
	last  time.Time // the last collection time
}

func (ct CollectTime) LastCollectTime() time.Time {
	return ct.last
}

func (ct CollectTime) FirstCollectTime() time.Time {
	return ct.first
}

func (ct *CollectTime) SetFirstCollectTime(first time.Time) {
	ct.first = first
}

func (ct *CollectTime) SetLastCollectTime(last time.Time) {
	ct.last = last
}

// Row holds a row of data from table_lock_waits_summary_by_table
type BaseObject struct {
	CollectTime
	ctx *context.Context
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
