package metrics

import "testing"

func TestAll(t *testing.T) {
	for _, m := range All() {
		if err := m.Def.Validate(); err != nil {
			t.Errorf("%s: %v", m.Def.Name, err)
		}
		// Result columns: key columns, then one per value column.
		if keys := len(m.Columns) - len(m.Def.Columns); keys < 1 {
			t.Errorf("%s: %d result columns for %d values", m.Def.Name, len(m.Columns), len(m.Def.Columns))
		}
		if len(m.Queries) == 0 || m.Entry == nil {
			t.Errorf("%s: no query or entry function", m.Def.Name)
		}
	}
}

func TestRequest(t *testing.T) {
	req := Request()
	if len(req.Candidates) != 2 || len(req.Variables) != 0 {
		t.Errorf("Request() = %+v, want 2 candidates and no variables", req)
	}
}
