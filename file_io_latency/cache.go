// Package key_value_cache provides an extremely simple string key
// to value cache.  This is used to reduce the number of lookups
// from MySQL filename to the equivalent table or logical name given
// the conversion routines to do this use many regexps and this is
// quite expensive.
package file_io_latency

import (
	"errors"
)

// kvCache provides a mapping from filename to table.schema etc.
// No protection against concurrent access is provided as this
// structure is expected to be accessed sequentially by a single
// go-routine.
type kvCache struct {
	cache           map[string]string
	readRequests    int
	servedFromCache int
	writeRequests   int
}

var (
	ErrNotFound = errors.New("Not found")
	cache       kvCache
)

// get will return the value in the cache if found
func (kvc *kvCache) get(key string) (result string, err error) {
	//	logger.Println("kvCache.Get(", key, ")")

	if kvc.cache == nil {
		//		logger.Println("kvCache.Get() kvc.cache is empty so enabling it")
		kvc.cache = make(map[string]string)
		kvc.readRequests = 0
		kvc.servedFromCache = 0
		kvc.writeRequests = 0
	}

	kvc.readRequests++

	if result, ok := kvc.cache[key]; ok {
		kvc.servedFromCache++
		//		logger.Println("Found: readRequests/servced_from_cache:", kvc.readRequests, kvc.servedFromCache)
		return result, nil
	}
	//	logger.Println("Not found: readRequests/servedFromCache:", kvc.readRequests, kvc.servedFromCache)

	return "", ErrNotFound
}

// put writes to cache and return the value saved.
func (kvc *kvCache) put(key, value string) string {
	//	logger.Println("kvCache.Put(", key, ",", value, ")")
	kvc.writeRequests++
	kvc.cache[key] = value

	return value
}

// statistics returns some staticts on read and write requests and
// the number of requests served from cache.
func (kvc *kvCache) statistics() (int, int, int) {
	return kvc.readRequests, kvc.servedFromCache, kvc.writeRequests
}
