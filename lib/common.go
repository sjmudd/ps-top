// Package lib includes several library routines for ps-top
package lib

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"

	"github.com/sjmudd/anonymiser"
)

const (
	copyright = "Copyright (C) 2014-2015 Simon J Mudd <sjmudd@pobox.com>"
	i1024_2   = 1024 * 1024
	i1024_3   = 1024 * 1024 * 1024
	i1024_4   = 1024 * 1024 * 1024 * 1024
)

var (
	myname string // program's name
)

// myround converts this floating value to the right width etc.
// There must be a function in Go to do this. Find it.
func myround(f float64, width, decimals int) string {
	format := "%" + fmt.Sprintf("%d", width) + "." + fmt.Sprintf("%d", decimals) + "f"
	return fmt.Sprintf(format, f)
}

// MyName returns the program's name based on a cleaned version of os.Args[0].
// Given this might be used a lot ensure we generate the value once and then
// cache the result.
func MyName() string {
	if myname == "" {
		myname = regexp.MustCompile(`.*/`).ReplaceAllLiteralString(os.Args[0], "")
	}

	return myname
}

// Copyright provides a copyright message for pstop
func Copyright() string {
	return copyright
}

// secToTime() converts a number of hours, minutes and seconds into hh:mm:ss format.
// e.g. 7384 = 2h 3m 4s, 7200 + 180 + 4
func secToTime(d uint64) string {
	hours := d / 3600                // integer value
	minutes := (d - hours*3600) / 60 // integer value
	seconds := d - hours*3600 - minutes*60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// FormatSeconds formats the seconds and is similar to secToTime() spaces if 0 and takes seconds as input.
func FormatSeconds(seconds uint64) string {
	if seconds == 0 {
		return "        "
	}
	return secToTime(seconds)
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
		return secToTime(picoseconds / 1000000000000)
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

// SginedFormatAccount formats a signed integer as per FormatAmount()
// FIXME - I've just copy pasted code but need to do this cleanly.
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

// Uptime provides a  usable form of uptime.
// Note: this doesn't return a string of a fixed size!
// Minimum value: 1s.
// Maximum value: 100d 23h 59m 59s (sort of).
func Uptime(uptime int) string {
	var result string

	days := uptime / 24 / 60 / 60
	hours := (uptime - days*86400) / 3600
	minutes := (uptime - days*86400 - hours*3600) / 60
	seconds := uptime - days*86400 - hours*3600 - minutes*60

	result = strconv.Itoa(seconds) + "s"

	if minutes > 0 {
		result = strconv.Itoa(minutes) + "m " + result
	}
	if hours > 0 {
		result = strconv.Itoa(hours) + "h " + result
	}
	if days > 0 {
		result = strconv.Itoa(days) + "d " + result
	}

	return result
}

// TableName returns the table name from the columns as '<schema>.<table>'
func TableName(schema, table string) string {
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
