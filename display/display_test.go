package display

import "testing"

func TestHostForDisplay(t *testing.T) {
	tests := []struct {
		bindAddress string
		want        string
	}{
		{"", "dbhost"},
		{"*", "dbhost"},
		{"0.0.0.0", "dbhost"},
		{"::", "dbhost"},
		{"127.0.0.1", "127.0.0.1"},
		{"10.0.0.5", "10.0.0.5"},
	}
	for _, tt := range tests {
		if got := hostForDisplay("dbhost", tt.bindAddress); got != tt.want {
			t.Errorf("hostForDisplay(%q, %q) = %q, want %q", "dbhost", tt.bindAddress, got, tt.want)
		}
	}
}
