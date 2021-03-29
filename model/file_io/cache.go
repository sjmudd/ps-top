package file_io

import (
	"errors"
)

// stringCache provides a mapping from filename to table.schema etc.
// No protection against concurrent access is provided as this
// structure is expected to be accessed sequentially by a single
// go-routine.
// Some counters are collected
type stringCache struct {
	cache map[string]string
}

var (
	cache stringCache
)

// get will return the value in the cache if found
func (sc *stringCache) get(key string) (result string, err error) {
	if sc.cache == nil {
		//	logger.Println("stringCache.get() sc is nil, enabling cache")
		sc.cache = make(map[string]string)
	}

	if result, ok := sc.cache[key]; ok {
		return result, nil
	}

	return "", errors.New("not found")
}

// put writes to cache and return the value saved.
func (sc *stringCache) put(key, value string) string {
	sc.cache[key] = value

	return value
}
