package presenter

import (
	"github.com/sjmudd/ps-top/v2/internal/formatter"
	"github.com/sjmudd/ps-top/v2/internal/metrics/memory"
)

// MemoryUsage shows memory in use now per instrument: bytes, allocation
// operations and allocation counts, current and highest.
var MemoryUsage = View[memory.Column]{
	Name:  "memory_usage",
	Title: "Memory Usage",
	Spec:  memory.Spec,
	Cells: [][]Cell[memory.Column]{
		{Value("CurBytes", memory.CurrentNumberOfBytesUsed, formatter.Amount, 10).LeftHeading(),
			ShareOfTotal("%", memory.CurrentNumberOfBytesUsed, 7),
			Value("High Bytes", memory.HighNumberOfBytesUsed, formatter.Amount, 11)},
		{Value("MemOps", memory.MemOps, formatter.Amount, 10).LeftHeading(), ShareOfTotal("%", memory.MemOps, 6)},
		{Value("CurAlloc", memory.CurrentCountUsed, formatter.Amount, 8),
			ShareOfTotal("%", memory.CurrentCountUsed, 7),
			Value("HiAlloc", memory.HighCountUsed, formatter.Amount, 9)},
		{Key[memory.Column]("Memory Area", 0)},
	},
	SortColumn: memory.CurrentNumberOfBytesUsed,
}
