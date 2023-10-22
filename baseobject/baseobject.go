// Package baseobject contains the library
// routines for base stuff of an object
package baseobject

import (
	"time"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/global"
	"github.com/sjmudd/ps-top/log"
	"github.com/sjmudd/ps-top/model/filter"
)

// BaseObject holds colllection times and a config
type BaseObject struct {
	cfg            *config.Config
	FirstCollected time.Time // the first collection time (for relative data)
	LastCollected  time.Time // the last collection time
}

// DatabaseFilter returns the config's DatabaseFilter()
func (o *BaseObject) DatabaseFilter() *filter.DatabaseFilter {
	return o.cfg.DatabaseFilter()
}

// SetConfig sets the config in this object which can be used later.
// - it should always be defined (!= nil)
func (o *BaseObject) SetConfig(cfg *config.Config) {
	if cfg == nil {
		log.Fatal("BaseObject.SetConfig(cfg) cfg should not be nil")
	}
	o.cfg = cfg
}

// Variables returns a pointer to the global variables
func (o BaseObject) Variables() *global.Variables {
	if o.cfg == nil {
		log.Fatal("BaseObject.Variables() o.cfg should not be nil")
	}
	return o.cfg.Variables()
}

// WantRelativeStats indicates whether we want relative stats or not
// - FIXME and optmise me away
func (o BaseObject) WantRelativeStats() bool {
	if o.cfg == nil {
		log.Fatal("BaseObject.WantRelativeStats(): o.cfg should not be nil")
		return false
	}
	return o.cfg.WantRelativeStats()
}
