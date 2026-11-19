package metric

import "testing"

func TestValidate(t *testing.T) {
	ok := Definition{
		Name: "io",
		Columns: []Column{
			{Name: "count_star", Kind: Counter},
			{Name: "sum_timer_wait", Kind: Counter},
		},
		ResetColumn: 1,
	}
	tests := []struct {
		name    string
		def     Definition
		wantErr bool
	}{
		{"valid", ok, false},
		{"no name", Definition{Columns: ok.Columns}, true},
		{"no columns", Definition{Name: "x"}, true},
		{"empty column name", Definition{Name: "x", Columns: []Column{{Kind: Counter}}}, true},
		{"duplicate column", Definition{Name: "x", Columns: []Column{{"a", Counter}, {"a", Counter}}}, true},
		{"invalid kind", Definition{Name: "x", Columns: []Column{{"a", Kind(9)}}}, true},
		{"reset out of range", Definition{Name: "x", Columns: []Column{{"a", Counter}}, ResetColumn: 1}, true},
		{"reset not counter", Definition{Name: "x", Columns: []Column{{"a", Gauge}}}, true},
	}
	for _, tt := range tests {
		if err := tt.def.Validate(); (err != nil) != tt.wantErr {
			t.Errorf("%s: Validate() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}
