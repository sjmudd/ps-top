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
	cache kvCache
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

	return "", errors.New("Not found")
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
