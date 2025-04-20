// Package utils includes several library routines for ps-top
package utils

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"

	"github.com/sjmudd/anonymiser"
)

const (
	// Copyright provide a copyright notice
	Copyright = "Copyright (C) 2014-2024 Simon J Mudd <sjmudd@pobox.com>"

	// Version returns the current application version
	Version = "1.1.14"

	i1024_2 = 1024 * 1024
	i1024_3 = 1024 * 1024 * 1024
	i1024_4 = 1024 * 1024 * 1024 * 1024
)

// ProgName returns the program's name based on a cleaned version of os.Args[0].
// Given this might be used a lot ensure we generate the value once and then
// cache the result.
var ProgName string

func init() {
	ProgName = regexp.MustCompile(`.*/`).ReplaceAllLiteralString(os.Args[0], "")
}

// DuplicateSlice is a slice duplicator using Go generics
func DuplicateSlice[T any](src []T) []T {
	dup := make([]T, len(src))
	copy(dup, src)
	return dup
}

// myround converts this floating value to the right width etc.
// There must be a function in Go to do this. Find it.
func myround(f float64, width, decimals int) string {
	format := "%" + fmt.Sprintf("%d", width) + "." + fmt.Sprintf("%d", decimals) + "f"
	return fmt.Sprintf(format, f)
}

// secToTime() converts a number of hours, minutes and seconds into hh:mm:ss format.
// e.g. 7384 = 2h 3m 4s, 7200 + 180 + 4
func secToTime(totalSeconds uint64) string {
	hours := totalSeconds / 3600                // integer value
	minutes := (totalSeconds - hours*3600) / 60 // integer value
	seconds := totalSeconds - hours*3600 - minutes*60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// FormatTime is based on sys.format_time. It
// formats to 10 characters including space and suffix.
// All values have 2 decimal places. Zero is returned as
// an empty string.
func FormatTime(picoseconds uint64) string {
	if picoseconds == 0 {
		return ""
	}
	if picoseconds >= 3600000000000000 {
		return myround(float64(picoseconds)/3600000000000000, 8, 2) + " h"
	}
	if picoseconds >= 60000000000000 {
		return myround(float64(picoseconds)/60000000000000, 8, 2) + " m"
	}
	if picoseconds >= 1000000000000 {
		return myround(float64(picoseconds)/1000000000000, 8, 2) + " s"
	}
	if picoseconds >= 1000000000 {
		return myround(float64(picoseconds)/1000000000, 7, 2) + " ms"
	}
	if picoseconds >= 1000000 {
		return myround(float64(picoseconds)/1000000, 7, 2) + " us"
	}
	if picoseconds >= 1000 {
		return myround(float64(picoseconds)/1000, 7, 2) + " ns"
	}
	return strconv.Itoa(int(picoseconds)) + " ps"
}

// FormatPct formats a floating point number as a percentage
// including the trailing % sign. Print the value as a %5.1f with
// a % suffix if there's a value.
// If the value is 0 print as 6 spaces.
// if the value is > 999.9 then show +++.+% to indicate an overflow.
func FormatPct(pct float64) string {
	var s string
	if pct < 0.0001 {
		s = "      "
	} else if pct > 999.9 {
		s = "+++.+%" // too large to fit! (probably a bug as we don't expect this value to be > 100.00)
	} else {
		s = fmt.Sprintf("%5.1f", 100.0*pct) + "%"
	}

	return s
}

// FormatAmount converts numbers to k = 1024 , M = 1024 x 1024, G = 1024 x 1024 x 1024, P = 1024x1024x1024x1024 and then formats them.
// For values = 0 return an empty string.
// For values < 1000 show 6,2 decimal places.
// For values >= 1000 show 6,1 decimal place.
func FormatAmount(amount uint64) string {
	var suffix string
	var formatted string
	var decimalAmount float64

	if amount == 0 {
		return ""
	}
	if amount <= 1024 {
		return strconv.Itoa(int(amount))
	}

	if amount > i1024_4 {
		suffix = "P"
		decimalAmount = float64(amount) / i1024_4
	} else if amount > i1024_3 {
		suffix = "G"
		decimalAmount = float64(amount) / i1024_3
	} else if amount > i1024_2 {
		suffix = "M"
		decimalAmount = float64(amount) / i1024_2
	} else if amount > 1024 {
		suffix = "k"
		decimalAmount = float64(amount) / 1024
	}

	if decimalAmount > 1000.0 {
		formatted = fmt.Sprintf("%6.1f %s", decimalAmount, suffix)
	} else {
		formatted = fmt.Sprintf("%6.2f %s", decimalAmount, suffix)
	}
	return formatted
}

// SignedFormatAmount formats a signed integer as per FormatAmount()
func SignedFormatAmount(amount int64) string {
	var suffix string
	var formatted string
	var decimalAmount float64

	if amount == 0 {
		return ""
	}
	if math.Abs(float64(amount)) <= 1024 {
		return strconv.Itoa(int(amount))
	}

	if math.Abs(float64(amount)) > i1024_4 {
		suffix = "P"
		decimalAmount = float64(amount) / i1024_4
	} else if math.Abs(float64(amount)) > i1024_3 {
		suffix = "G"
		decimalAmount = float64(amount) / i1024_3
	} else if math.Abs(float64(amount)) > i1024_2 {
		suffix = "M"
		decimalAmount = float64(amount) / i1024_2
	} else if math.Abs(float64(amount)) > 1024 {
		suffix = "k"
		decimalAmount = float64(amount) / 1024
	}

	if math.Abs(decimalAmount) > 1000.0 {
		formatted = fmt.Sprintf("%6.1f %s", decimalAmount, suffix)
	} else {
		formatted = fmt.Sprintf("%6.2f %s", decimalAmount, suffix)
	}
	return formatted
}

// FormatCounter formats a counter like an Amount but is tighter in space
func FormatCounter(counter int, width int) string {
	if counter == 0 {
		pattern := "%" + fmt.Sprintf("%d", width) + "s"
		return fmt.Sprintf(pattern, " ")
	}
	pattern := "%" + fmt.Sprintf("%d", width) + "d"
	return fmt.Sprintf(pattern, counter)
}

// Divide divides a by b except if b is 0 in which case we return 0.
func Divide(a uint64, b uint64) float64 {
	if b == 0 {
		return float64(0)
	}
	return float64(a) / float64(b)
}

// SignedDivide divides a by b except if b is 0 in which case we return 0.
func SignedDivide(a int64, b int64) float64 {
	if b == 0 {
		return float64(0)
	}
	return float64(a) / float64(b)
}

// QualifiedTableName returns the anonymised qualified table name from the columns as '<schema>.<table>'
func QualifiedTableName(schema, table string) string {
	schema = anonymiser.Anonymise("schema", schema)
	table = anonymiser.Anonymise("table", table)

	var name string
	if len(schema) > 0 {
		name += schema
	}
	if len(name) > 0 {
		if len(table) > 0 {
			name += "." + table
		}
	} else {
		if len(table) > 0 {
			name += table
		}
	}
	return name
}
