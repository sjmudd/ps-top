// Package baseobject contains the library
// routines for base stuff of an object
package baseobject

import (
	"log"
	"time"

	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/global"
)

// Row holds a row of data from table_lock_waits_summary_by_table
type BaseObject struct {
	intialCollectTime time.Time // the initial collection time (for relative data)
	lastCollectTime   time.Time // the last collection time
	ctx               *context.Context
}

func (o BaseObject) LastCollectTime() time.Time {
	return o.lastCollectTime
}

// SetNow records the time the data was collected (now)
func (o *BaseObject) SetLastCollectTimeNow() {
	o.lastCollectTime = time.Now()
}

func (o BaseObject) InitialCollectTime() time.Time {
	return o.intialCollectTime
}

func (o *BaseObject) SetInitialCollectTime(initial time.Time) {
	o.intialCollectTime = initial
}

// SetNow records the time the data was collected (now)
func (o *BaseObject) SetInitialCollectTimeNow() {
	o.intialCollectTime = time.Now()
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
