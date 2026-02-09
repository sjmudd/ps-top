//nolint:revive
package common

import (
	"database/sql"
	"log"
)

// SubtractByName removes initial values from rows where there's a matching name.
// It is a small generic helper to reduce duplicated subtract logic across model
// packages. Callers provide a way to obtain the name for a row and a subtract
// implementation for the concrete row type.
//
// We use two type parameters: T is the element type and S is the slice type
// (e.g. `Rows`) with a type approximation so callers can pass a named slice
// type like `Rows` without needing to expose the underlying element type.
func SubtractByName[T any, S ~[]T](rows *S, initial S, nameOf func(T) string, subtract func(*T, T)) {
	initialByName := make(map[string]int)

	// build map of initial rows by name
	for i := range initial {
		initialByName[nameOf(initial[i])] = i
	}

	// iterate the target rows and subtract matching initial values
	for i := range *rows {
		name := nameOf((*rows)[i])
		if initialIndex, ok := initialByName[name]; ok {
			subtract(&(*rows)[i], initial[initialIndex])
		}
	}
}

// SubtractCounts subtracts two countable values (sum and count) safely.
// If the left-hand sum is less than the other sum a warning is logged and
// no subtraction is performed. The helper accepts opaque row values which
// are logged on warning to aid debugging.
func SubtractCounts(sum *uint64, count *uint64, otherSum, otherCount uint64, row any, other any) {
	if *sum >= otherSum {
		*sum -= otherSum
		*count -= otherCount
	} else {
		log.Println("WARNING: SubtractCounts() - subtraction problem! (not subtracting)")
		log.Println("row=", row)
		log.Println("other=", other)
	}
}

// NeedsRefresh compares two total SumTimerWait values and returns true if
// the first appears to be "newer" (i.e. larger) than the second. Extracted
// as a small helper to reduce duplicated needsRefresh implementations in
// multiple model packages.
func NeedsRefresh(firstTotal, otherTotal uint64) bool {
	return firstTotal > otherTotal
}

// Collect is a small generic helper that consumes sql.Rows using a caller
// provided scanner closure. The scanner should scan the current row from the
// provided *sql.Rows and return the concrete row value. Collect handles the
// rows.Next loop, rows.Err() check and rows.Close() cleanup to avoid
// duplicating that logic across model packages.
func Collect[T any](rows *sql.Rows, scanner func() (T, error)) []T {
	var t []T

	for rows.Next() {
		r, err := scanner()
		if err != nil {
			log.Fatal(err)
		}
		t = append(t, r)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	_ = rows.Close()

	return t
}
