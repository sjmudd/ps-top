package file_summary_by_instance

import (
	"errors"

	//	"github.com/sjmudd/ps-top/lib"
)

// provide a mapping from filename to table.schema etc

var (
	mapped_name    map[string]string
	total, matched int
)

func init() {
	// setup on startup
	mapped_name = make(map[string]string)
}

func get_from_cache(key string) (result string, err error) {
	total++
	if result, ok := mapped_name[key]; ok {
		matched++
		//		lib.Logger.Println("matched/total:", matched, total)
		return result, nil
	} else {
		//		lib.Logger.Println("matched/total:", matched, total)
		return "", errors.New("Not found")
	}
}

func save_to_cache(key, value string) string {
	mapped_name[key] = value
	return value
}
