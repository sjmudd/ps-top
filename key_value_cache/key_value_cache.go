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

// create a new KeyValueCache entry
func NewKeyValueCache() KeyValueCache {
	lib.Logger.Println("KeyValueCache()")
	var kvc KeyValueCache

	return kvc
}

// return value if found
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

// write to cache and return value
func (kvc *KeyValueCache) Put(key, value string) string {
	lib.Logger.Println("KeyValueCache.Put(", key, ",", value, ")")
	kvc.cache[key] = value
	return value
}

func (kvc *KeyValueCache) Statistics() (int, int, int) {
	return kvc.read_requests, kvc.served_from_cache, kvc.write_requests
}
