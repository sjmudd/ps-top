package presenter

import (
	"github.com/sjmudd/ps-top/internal/formatter"
	"github.com/sjmudd/ps-top/internal/metrics/tableio"
)

// TableIOLatency shows table I/O wait time per table.
var TableIOLatency = View[tableio.Column]{
	Title: "Table I/O Latency",
	Spec:  tableio.Spec,
	Cells: [][]Cell[tableio.Column]{
		{Value("Latency", tableio.SumTimerWait, formatter.Time, 10), ShareOfTotal("%", tableio.SumTimerWait, 6)},
		{ShareOf("Read", tableio.SumTimerRead, tableio.SumTimerWait, 6), ShareOf("Write", tableio.SumTimerWrite, tableio.SumTimerWait, 6)},
		{ShareOf("Fetch", tableio.SumTimerFetch, tableio.SumTimerWait, 6), ShareOf("Insert", tableio.SumTimerInsert, tableio.SumTimerWait, 6),
			ShareOf("Update", tableio.SumTimerUpdate, tableio.SumTimerWait, 6), ShareOf("Delete", tableio.SumTimerDelete, tableio.SumTimerWait, 6)},
		{Key[tableio.Column]("Table Name", 0)},
	},
	SortColumn: tableio.SumTimerWait,
}

// TableIOOps shows table I/O operation counts per table.
var TableIOOps = View[tableio.Column]{
	Title: "Table I/O Ops",
	Spec:  tableio.Spec,
	Cells: [][]Cell[tableio.Column]{
		{Value("Ops", tableio.CountStar, formatter.Count, 10), ShareOfTotal("%", tableio.CountStar, 6)},
		{ShareOf("Read", tableio.CountRead, tableio.CountStar, 6), ShareOf("Write", tableio.CountWrite, tableio.CountStar, 6)},
		{ShareOf("Fetch", tableio.CountFetch, tableio.CountStar, 6), ShareOf("Insert", tableio.CountInsert, tableio.CountStar, 6),
			ShareOf("Update", tableio.CountUpdate, tableio.CountStar, 6), ShareOf("Delete", tableio.CountDelete, tableio.CountStar, 6)},
		{Key[tableio.Column]("Table Name", 0)},
	},
	SortColumn: tableio.CountStar,
}
