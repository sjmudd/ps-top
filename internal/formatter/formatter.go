// Package formatter turns values into display strings: times in picoseconds,
// counts, amounts and percentages.
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

// Count formats c in decimal. Zero is "".
func Count(c uint64) string {
	if c == 0 {
		return ""
	}
	return strconv.FormatUint(c, 10)
}

// Amount formats a size or count with a 1024-based suffix (k, M, G, T, P):
// up to 1024 as is, above with two decimals (one above 1000), e.g. "1023",
// "  1.00 k", "1024.0 k". Zero is "".
func Amount(a uint64) string {
	if a == 0 {
		return ""
	}
	if a <= 1024 {
		return strconv.FormatUint(a, 10)
	}
	const k = 1024
	var div float64
	var suffix string
	switch {
	case a > k*k*k*k*k:
		div, suffix = k*k*k*k*k, "P"
	case a > k*k*k*k:
		div, suffix = k*k*k*k, "T"
	case a > k*k*k:
		div, suffix = k*k*k, "G"
	case a > k*k:
		div, suffix = k*k, "M"
	default:
		div, suffix = k, "k"
	}
	f := float64(a) / div
	if f > 1000 {
		return fmt.Sprintf("%6.1f %s", f, suffix)
	}
	return fmt.Sprintf("%6.2f %s", f, suffix)
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
