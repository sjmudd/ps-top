package formatter

import "testing"

func TestTime(t *testing.T) {
	for _, tt := range []struct {
		ps   uint64
		want string
	}{
		{0, ""},
		{1, "1 ps"},
		{999, "999 ps"},
		{1000, "   1.00 ns"},
		{999000, " 999.00 ns"},
		{1000000, "   1.00 us"},
		{999000000, " 999.00 us"},
		{1000000000, "   1.00 ms"},
		{999000000000, " 999.00 ms"},
		{1000000000000, "    1.00 s"},
		{59000000000000, "   59.00 s"},
		{60000000000000, "    1.00 m"},
		{3540000000000000, "   59.00 m"},
		{3600000000000000, "    1.00 h"},
		{3600000000000000 * 24, "   24.00 h"},
	} {
		if got := Time(tt.ps); got != tt.want {
			t.Errorf("Time(%d) = %q, want %q", tt.ps, got, tt.want)
		}
	}
}

func TestPct(t *testing.T) {
	for _, tt := range []struct {
		f    float64
		want string
	}{
		{0, "      "},
		{0.00009, "      "},
		{0.0049, "  0.5%"},
		{0.049, "  4.9%"},
		{0.5, " 50.0%"},
		{1, "100.0%"},
		{9.999, "999.9%"},
		{10.001, "+++.+%"},
	} {
		if got := Pct(tt.f); got != tt.want {
			t.Errorf("Pct(%v) = %q, want %q", tt.f, got, tt.want)
		}
	}
}

func TestCount(t *testing.T) {
	if got := Count(0); got != "" {
		t.Errorf("Count(0) = %q, want empty", got)
	}
	if got := Count(42); got != "42" {
		t.Errorf("Count(42) = %q", got)
	}
}

func TestAmount(t *testing.T) {
	const k = 1024
	for _, tt := range []struct {
		a    uint64
		want string
	}{
		// as the old utils.FormatAmount
		{0, ""},
		{1, "1"},
		{k - 1, "1023"},
		{k, "1024"},
		{k + 1, "  1.00 k"},
		{k * k, "1024.0 k"},
		{k * k * k, "1024.0 M"},
		{k * k * k * k, "1024.0 G"},
		// the old code showed these as P
		{k*k*k*k + 1, "  1.00 T"},
		{2 * k * k * k * k, "  2.00 T"},
		{k * k * k * k * k, "1024.0 T"},
		{k*k*k*k*k + 1, "  1.00 P"},
	} {
		if got := Amount(tt.a); got != tt.want {
			t.Errorf("Amount(%d) = %q, want %q", tt.a, got, tt.want)
		}
	}
}

// TestSeconds checks the old program's user view format, including units
// left out when the result would be over 10 characters.
func TestSeconds(t *testing.T) {
	for d, want := range map[uint64]string{
		0:                        "",
		45:                       "45s",
		70:                       "1m 10s",
		3600:                     "1h 0m 0s",
		2*3600 + 3*60 + 4:        "2h 3m 4s",
		12*3600 + 34*60 + 56:     "12h 34m", // "12h 34m 56s" is 11 characters
		86400 + 2*3600 + 3*60:    "1d 2h 3m",
		100*86400 + 23*3600 + 59: "100d 23h", // "100d 23h 0m" is 11 characters
	} {
		if got := Seconds(d); got != want {
			t.Errorf("Seconds(%d) = %q, want %q", d, got, want)
		}
	}
}

func TestDivide(t *testing.T) {
	if Divide(1, 0) != 0 || Divide(1, 4) != 0.25 {
		t.Errorf("Divide: got %v and %v, want 0 and 0.25", Divide(1, 0), Divide(1, 4))
	}
}
