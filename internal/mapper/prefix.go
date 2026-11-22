package mapper

import "strings"

// TrimPrefix returns a mapper that removes prefix from a key, e.g.
// TrimPrefix("stage/sql/") maps "stage/sql/Sending data" to "Sending data".
// A key without the prefix is returned as it is.
func TrimPrefix(prefix string) func(string) string {
	return func(k string) string { return strings.TrimPrefix(k, prefix) }
}
