package file_io_latency

import (
	"errors"
)

// provide a mapping from filename to table.schema etc

var (
	mappedName     map[string]string
	total, matched int
)

func init() {
	// setup on startup
	mappedName = make(map[string]string)
}

func getFromCache(key string) (result string, err error) {
	total++
	if result, ok := mappedName[key]; ok {
		matched++
		//		logger.Println("matched/total:", matched, total)
		return result, nil
	}
	//		logger.Println("matched/total:", matched, total)
	return "", errors.New("Not found")
}

func saveToCache(key, value string) string {
	mappedName[key] = value
	return value
}
