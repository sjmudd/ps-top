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

// ShortTime formats picoseconds in at most 8 characters with no spaces.
// Below a minute: one decimal and a unit, e.g. "844.0us", "1.1ms", "59.1s".
// From a minute: whole seconds (cut, not rounded) as "9m54s" or
// "1h23m10s", without seconds where that does not fit, e.g. "23h59m", and
// from a day "1d13h5m", without seconds. Zero is "".
func ShortTime(ps uint64) string {
	const second = 1000000000000
	if ps >= 60*second {
		s := ps / second
		d, h, m := s/86400, s/3600%24, s/60%60
		s %= 60
		switch {
		case d > 0:
			return fmt.Sprintf("%dd%dh%dm", d, h, m)
		case h > 0:
			if t := fmt.Sprintf("%dh%dm%ds", h, m, s); len(t) <= 8 {
				return t
			}
			return fmt.Sprintf("%dh%dm", h, m)
		}
		return fmt.Sprintf("%dm%ds", m, s)
	}
	units := []struct {
		ps   uint64
		name string
	}{
		{second, "s"}, {1000000000, "ms"}, {1000000, "us"}, {1000, "ns"},
	}
	if ps == 0 {
		return ""
	}
	for _, u := range units {
		if ps >= u.ps {
			return fmt.Sprintf("%.1f%s", float64(ps)/float64(u.ps), u.name)
		}
	}
	return strconv.FormatUint(ps, 10) + "ps"
}

// round formats f with two decimals in width characters.
func round(f float64, width int) string {
	return fmt.Sprintf("%*.2f", width, f)
}

// Seconds formats a duration in seconds in at most 10 characters, as v1's user
// view: "45s", "1m 10s", "2h 3m 4s", "1d 2h 3m". A unit that
// does not fit is left out, e.g. "12h 34m" not "12h 34m 56s". Zero is "".
func Seconds(d uint64) string {
	if d == 0 {
		return ""
	}
	days := d / 86400
	hours := d % 86400 / 3600
	minutes := d % 3600 / 60
	seconds := d % 60
	switch {
	case days > 0:
		if s := fmt.Sprintf("%dd %dh %dm", days, hours, minutes); len(s) <= 10 {
			return s
		}
		return fmt.Sprintf("%dd %dh", days, hours)
	case hours > 0:
		if s := fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds); len(s) <= 10 {
			return s
		}
		return fmt.Sprintf("%dh %dm", hours, minutes)
	case minutes > 0:
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
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
