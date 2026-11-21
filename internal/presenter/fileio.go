package presenter

import (
	"github.com/sjmudd/ps-top/internal/formatter"
	"github.com/sjmudd/ps-top/internal/metrics/fileio"
)

// FileIOLatency shows file I/O wait time, bytes and operations per file,
// with file names mapped to tables or labels by mapKey (built from the
// server's settings, so the view is made after the probe).
func FileIOLatency(mapKey func(string) string) View[fileio.Column] {
	return View[fileio.Column]{
		Title:  "File I/O Latency",
		Spec:   fileio.Spec,
		MapKey: mapKey,
		Cells: [][]Cell[fileio.Column]{
			{Value("Latency", fileio.SumTimerWait, formatter.Time, 10), ShareOfTotal("%", fileio.SumTimerWait, 6)},
			{ShareOf("Read", fileio.SumTimerRead, fileio.SumTimerWait, 6), ShareOf("Write", fileio.SumTimerWrite, fileio.SumTimerWait, 6),
				ShareOf("Misc", fileio.SumTimerMisc, fileio.SumTimerWait, 6)},
			{Value("Rd bytes", fileio.SumNumberOfBytesRead, formatter.Amount, 8),
				Value("Wr bytes", fileio.SumNumberOfBytesWrite, formatter.Amount, 8)},
			{Value("Ops", fileio.CountStar, formatter.Amount, 8), ShareOf("R Ops", fileio.CountRead, fileio.CountStar, 6),
				ShareOf("W Ops", fileio.CountWrite, fileio.CountStar, 6), ShareOf("M Ops", fileio.CountMisc, fileio.CountStar, 6)},
			{Key[fileio.Column]("Table Name", 0)},
		},
		SortColumn: fileio.SumTimerWait,
	}
}
