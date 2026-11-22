package metric

import "testing"

func TestValidate(t *testing.T) {
	diff := []MetricType{DifferenceValues}
	ok := Definition{
		Name: "io",
		Columns: []Column{
			{Name: "count_star", Kind: Counter},
			{Name: "sum_timer_wait", Kind: Counter},
		},
		Types:       diff,
		ResetColumn: 1,
	}
	a := func(k Kind) []Column { return []Column{{Name: "a", Kind: k}} }
	tests := []struct {
		name    string
		def     Definition
		wantErr bool
	}{
		{"valid", ok, false},
		{"no name", Definition{Columns: ok.Columns, Types: diff}, true},
		{"no columns", Definition{Name: "x", Types: diff}, true},
		{"empty column name", Definition{Name: "x", Columns: []Column{{Kind: Counter}}, Types: diff}, true},
		{"duplicate column", Definition{Name: "x", Columns: append(a(Counter), a(Counter)...), Types: diff}, true},
		{"invalid kind", Definition{Name: "x", Columns: a(Kind(9)), Types: diff}, true},
		{"no types", Definition{Name: "x", Columns: a(Counter)}, true},
		{"invalid type", Definition{Name: "x", Columns: a(Counter), Types: []MetricType{9}}, true},
		{"duplicate type", Definition{Name: "x", Columns: a(Counter), Types: []MetricType{CurrentValue, CurrentValue}}, true},
		{"reset out of range", Definition{Name: "x", Columns: a(Counter), Types: diff, ResetColumn: 1}, true},
		{"reset not counter", Definition{Name: "x", Columns: a(Gauge), Types: diff}, true},
		// CurrentValue alone does not detect resets, so needs no counter.
		{"current, gauges only", Definition{Name: "x", Columns: a(Gauge), Types: []MetricType{CurrentValue}}, false},
		{"current and difference, gauges only", Definition{Name: "x", Columns: a(Gauge), Types: []MetricType{CurrentValue, DifferenceValues}}, true},
	}
	for _, tt := range tests {
		if err := tt.def.Validate(); (err != nil) != tt.wantErr {
			t.Errorf("%s: Validate() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestMetricType(t *testing.T) {
	for _, tt := range []struct {
		t               MetricType
		snapshots, ring int
	}{
		{CurrentValue, 1, 1},
		{DifferenceValues, 2, 11},
	} {
		if got := tt.t.Snapshots(); got != tt.snapshots {
			t.Errorf("%v.Snapshots() = %d, want %d", tt.t, got, tt.snapshots)
		}
		if got := tt.t.RingSize(10); got != tt.ring {
			t.Errorf("%v.RingSize(10) = %d, want %d", tt.t, got, tt.ring)
		}
	}
}
