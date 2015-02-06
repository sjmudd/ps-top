// key_value_cache provides an extremely simple string key to value cache.
// This is used to reduce the number of lookups from MySQL filename
// to the equivalent table or logical name given the conversion
// routines to do this use many regexps and this is quite expensive.
package key_value_cache

import (
	"errors"

	"github.com/sjmudd/pstop/lib"
)

// provide a mapping from filename to table.schema etc
type KeyValueCache struct {
	cache                                            map[string]string
	read_requests, served_from_cache, write_requests int
}

// Create a new KeyValueCache entry.
func NewKeyValueCache() KeyValueCache {
	lib.Logger.Println("KeyValueCache()")

	return KeyValueCache{}
}

// Given a lookup key return the value if found.
func (kvc *KeyValueCache) Get(key string) (result string, err error) {
	lib.Logger.Println("KeyValueCache.Get(", key, ")")
	if kvc.cache == nil {
		lib.Logger.Println("KeyValueCache.Get() kvc.cache is empty so enabling it")
		kvc.cache = make(map[string]string)
		kvc.read_requests = 0
		kvc.served_from_cache = 0
		kvc.write_requests = 0
	}

	kvc.read_requests++

	if result, ok := kvc.cache[key]; ok {
		kvc.served_from_cache++
		lib.Logger.Println("Found: read_requests/servced_from_cache:", kvc.read_requests, kvc.served_from_cache)
		return result, nil
	} else {
		lib.Logger.Println("Not found: read_requests/served_from_cache:", kvc.read_requests, kvc.served_from_cache)
		return "", errors.New("Not found")
	}
}

// Write to cache and return the value saved.
func (kvc *KeyValueCache) Put(key, value string) string {
	lib.Logger.Println("KeyValueCache.Put(", key, ",", value, ")")
	kvc.cache[key] = value
	return value
}

// Provide some staticts on read and write requests and the number
// of requests served from cache.
func (kvc *KeyValueCache) Statistics() (int, int, int) {
	return kvc.read_requests, kvc.served_from_cache, kvc.write_requests
}
