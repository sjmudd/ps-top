package transform

import (
	"strings"
	"testing"
	"time"

	"github.com/sjmudd/ps-top/internal/filemap"
	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/snapshot"
)

var fileIO = metric.Definition{
	Name: "file_io",
	Columns: []metric.Column{
		{Name: "count_star", Kind: metric.Counter},
		{Name: "sum_timer_wait", Kind: metric.Counter},
	},
	ResetColumn: 1,
}

func TestAggregateFileIO(t *testing.T) {
	m, err := filemap.New(filemap.Settings{DataDir: "/var/lib/mysql/"})
	if err != nil {
		t.Fatal(err)
	}
	raw := []Entry{
		{"/var/lib/mysql/shop/orders#p#p2023.ibd", []uint64{100, 5000}},
		{"/var/lib/mysql/shop/orders#p#p2024.ibd", []uint64{300, 9000}},
		{"/var/lib/mysql/shop/customers.ibd", []uint64{200, 4000}},
		{"/var/lib/mysql/#innodb_temp/temp_1.ibt", []uint64{50, 1000}},
		{"/var/lib/mysql/#innodb_temp/temp_2.ibt", []uint64{70, 1500}},
		{"/var/lib/mysql/ibtmp1", []uint64{7, 70}},
	}
	rows, err := Aggregate(fileIO, raw, m.Map)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		key    string
		values [2]uint64
	}{
		{"shop.orders", [2]uint64{400, 14000}},
		{"shop.customers", [2]uint64{200, 4000}},
		{"<temp_table>", [2]uint64{120, 2500}},
		{"<ibtmp>", [2]uint64{7, 70}},
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		if rows[i].Key.Value() != w.key || rows[i].Values[0] != w.values[0] || rows[i].Values[1] != w.values[1] {
			t.Errorf("row %d = %s %v, want %s %v", i, rows[i].Key.Value(), rows[i].Values, w.key, w.values)
		}
	}
	// The result is a valid snapshot: no duplicate keys.
	if _, err := snapshot.New(time.Now(), 2, rows); err != nil {
		t.Errorf("snapshot.New: %v", err)
	}
}

func TestAggregateCopiesValues(t *testing.T) {
	raw := []Entry{{"a", []uint64{1, 2}}}
	rows, err := Aggregate(fileIO, raw, strings.ToUpper)
	if err != nil {
		t.Fatal(err)
	}
	raw[0].Values[0] = 99
	if rows[0].Values[0] != 1 || rows[0].Key.Value() != "A" {
		t.Errorf("row = %s %v", rows[0].Key.Value(), rows[0].Values)
	}
}

func TestAggregateNilMapKey(t *testing.T) {
	raw := []Entry{{"db.a", []uint64{1, 2}}, {"db.b", []uint64{3, 4}}, {"db.a", []uint64{5, 6}}}
	entries, err := Aggregate(fileIO, raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Key.Value() != "db.a" || entries[0].Values[0] != 6 || entries[1].Key.Value() != "db.b" {
		t.Errorf("entries = %v", entries)
	}
}

func TestAggregateErrors(t *testing.T) {
	id := func(s string) string { return s }
	if _, err := Aggregate(fileIO, []Entry{{"a", []uint64{1}}}, id); err == nil {
		t.Error("wrong width: no error")
	}
	if _, err := Aggregate(metric.Definition{}, nil, id); err == nil {
		t.Error("invalid definition: no error")
	}
}
