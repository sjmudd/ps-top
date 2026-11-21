// Package formatter turns values into fixed-width display strings: times in
// picoseconds, counts and percentages.
package formatter

import (
	"fmt"
	"strconv"
)

// Time formats picoseconds with a unit (ps, ns, us, ms, s, m, h). Zero is "".
func Time(ps uint64) string {
	if ps == 0 {
		return ""
	}
	switch {
	case ps >= 3600000000000000:
		return round(float64(ps)/3600000000000000, 8) + " h"
	case ps >= 60000000000000:
		return round(float64(ps)/60000000000000, 8) + " m"
	case ps >= 1000000000000:
		return round(float64(ps)/1000000000000, 8) + " s"
	case ps >= 1000000000:
		return round(float64(ps)/1000000000, 7) + " ms"
	case ps >= 1000000:
		return round(float64(ps)/1000000, 7) + " us"
	case ps >= 1000:
		return round(float64(ps)/1000, 7) + " ns"
	}
	return strconv.FormatUint(ps, 10) + " ps"
}

// round formats f with two decimals in width characters.
func round(f float64, width int) string {
	return fmt.Sprintf("%*.2f", width, f)
}

// Count formats c right-aligned in width characters. Zero is blank.
func Count(c uint64, width int) string {
	if c == 0 {
		return fmt.Sprintf("%*s", width, "")
	}
	return fmt.Sprintf("%*d", width, c)
}

// Pct formats a fraction as a percentage in 6 characters, e.g. " 50.0%".
// Below 0.0001 it is blank; 10 (1000%) or more shows "+++.+%".
func Pct(f float64) string {
	switch {
	case f < 0.0001:
		return "      "
	case f >= 10.0:
		return "+++.+%"
	}
	return fmt.Sprintf("%5.1f%%", 100*f)
}

// Divide returns a / b, or 0 if b is 0.
func Divide(a, b uint64) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}
