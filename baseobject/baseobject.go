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
	config         *config.Config
	FirstCollected time.Time // the first collection time (for relative data)
	LastCollected  time.Time // the last collection time
}

// DatabaseFilter returns the config's DatabaseFilter()
func (o *BaseObject) DatabaseFilter() *filter.DatabaseFilter {
	return o.config.DatabaseFilter()
}

// SetConfig sets the config in this object which can be used later.
// - it should always be defined (!= nil)
func (o *BaseObject) SetConfig(config *config.Config) {
	if config == nil {
		log.Fatal("BaseObject.SetConfig(config) config should not be nil")
	}
	o.config = config
}

// Variables returns a pointer to the global variables
func (o BaseObject) Variables() *global.Variables {
	if o.config == nil {
		log.Fatal("BaseObject.Variables() o.config should not be nil")
	}
	return o.config.Variables()
}

// WantRelativeStats indicates whether we want relative stats or not
// - FIXME and optmise me away
func (o BaseObject) WantRelativeStats() bool {
	if o.config == nil {
		log.Fatal("BaseObject.WantRelativeStats(): o.config should not be nil")
		return false
	}
	return o.config.WantRelativeStats()
}
