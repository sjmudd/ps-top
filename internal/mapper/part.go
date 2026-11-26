package mapper

import "github.com/sjmudd/ps-top/v2/internal/snapshot"

// Part returns a mapper that keeps only part i of a key of several parts,
// e.g. Part(0) maps a connection's key (user, host, db, ID) to its user, so
// a view shows one row per user. A key with no part i maps to "".
func Part(i int) func(string) string {
	return func(k string) string {
		if parts := snapshot.KeyParts(k); i < len(parts) {
			return parts[i]
		}
		return ""
	}
}
