package display

import (
	"time"
)

// HelpType is a help information provided by the generic interface
type HelpType struct{}

func (h HelpType) Description() string         { return "Help" }
func (h HelpType) Headings() string            { return "                   ---=== Help for using ps-top ===---" }
func (h HelpType) FirstCollectTime() time.Time { return time.Now() }
func (h HelpType) LastCollectTime() time.Time  { return time.Now() }
func (h HelpType) RowContent() []string {
	return []string{
		"",
		"                                ps-top",
		"                                ------",
		"",
		"A program to show the top I/O information by accessing information from the",
		"performance_schema schema. Ideas based on mysql-sys.",
		"",
		"Keys:",
		"   - - reduce the poll interval by 1 second (minimum 1 second)",
		"   + - increase the poll interval by 1 second",
		"   h/? - this help screen",
		"   q - quit",
		"   s - sort differently (where enabled) - sorts on a different column",
		"   t - toggle between showing time since resetting statistics or since P_S data was collected",
		"   z - reset statistics",
		"   <tab> or <right arrow> - change display modes between: latency, ops,",
		"                            file I/O, lock and user modes",
		"   <left arrow> - change display modes to the previous screen (see above)",
		"",
		"Press h to return to main screen",
	}
}
func (h HelpType) TotalRowContent() string { return "" }
func (h HelpType) EmptyRowContent() string { return "" }
func (h HelpType) HaveRelativeStats() bool { return false }

var Help HelpType // empty initialisation should be ok for providing help
