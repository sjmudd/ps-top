package presenter

import (
	"github.com/sjmudd/ps-top/v2/internal/formatter"
	"github.com/sjmudd/ps-top/v2/internal/metrics/fileio"
	"github.com/sjmudd/ps-top/v2/internal/snapshot"
)

// FileIOLatency shows file I/O wait time, bytes and operations per file,
// with file names mapped to tables or labels by mapKey (built from the
// server's settings, so the view is made after the probe). A table is a key
// of two parts, schema and table, which are anonymised with --anonymise. A
// label or a path is one part and is always shown as it is. With
// --database-filter, a table is shown only if its schema is one of those
// named, and a label or a path is always shown.
func FileIOLatency(mapKey func(string) string) View[fileio.Column] {
	v := View[fileio.Column]{
		Name:      "file_io_latency",
		Title:     "File I/O Latency",
		Spec:      fileio.Spec,
		MapKey:    mapKey,
		Anonymise: anonymiseTable,
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
	if mapKey != nil {
		v.Schema = func(raw string) string { return tableSchema(mapKey(raw)) }
	}
	return v
}

// tableSchema returns the schema of a key of schema and table, else "".
func tableSchema(key string) string {
	if parts := snapshot.KeyParts(key); len(parts) == 2 {
		return parts[0]
	}
	return ""
}
