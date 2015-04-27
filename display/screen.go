package display

import (
	"fmt"
	"time"

	"github.com/sjmudd/pstop/i_s/processlist"
	"github.com/sjmudd/pstop/lib"
	"github.com/sjmudd/pstop/p_s/ps_table"
	tiwsbt "github.com/sjmudd/pstop/p_s/table_io_waits_summary_by_table"
	"github.com/sjmudd/pstop/screen"
)

type ScreenDisplay struct {
	DisplayHeading // embedded
	screen         *screen.TermboxScreen
	limit          int
}

func (s *ScreenDisplay) display(t GenericObject) {
	var top_line string

	top_line_start := s.Myname + " " + s.Version + " - " + now_hhmmss() + " " + s.Hostname + " / " + s.MysqlVersion + ", up " + fmt.Sprintf("%-16s", lib.Uptime(s.Uptime))

	if s.WantRelativeStats {
		top_line = top_line_start + " [REL] " + fmt.Sprintf("%.0f seconds", s.rel_time(t.Last()))
	} else {
		top_line = top_line_start + " [ABS]             "
	}
	s.screen.PrintAt(0, 0, top_line)
	s.screen.PrintAt(0, 1, t.Description())
	s.screen.BoldPrintAt(0, 2, t.Headings())

	max_rows := s.screen.Height() - 4
	last_row := s.screen.Height() - 1
	row_content := t.RowContent(max_rows)

	// print out rows
	for k := range row_content {
		y := 3 + k
		s.screen.PrintAt(0, y, row_content[k])
		s.screen.ClearLine(len(row_content[k]), y)
	}
	// print out empty rows
	for k := len(row_content); k < max_rows; k++ {
		y := 3 + k
		if y < last_row {
			s.screen.PrintAt(0, y, t.EmptyRowContent())
		}
	}

	// print out the totals at the bottom
	total := t.TotalRowContent()
	s.screen.BoldPrintAt(0, last_row, total)
	s.screen.ClearLine(len(total), last_row)
}

func (s *ScreenDisplay) SetScreen(screen *screen.TermboxScreen) {
	s.screen = screen
}

func (s *ScreenDisplay) SetLimit(limit int) {
	s.limit = limit
}

// print out to stdout the IO values
func (s *ScreenDisplay) DisplayIO(fsbi ps_table.Tabler) {
	s.display(fsbi)
}

func (s *ScreenDisplay) DisplayLocks(tlwsbt ps_table.Tabler) {
	s.display(tlwsbt)
}

func (s *ScreenDisplay) DisplayMutex(ewsgben ps_table.Tabler) {
	s.display(ewsgben)
}

func (s *ScreenDisplay) DisplayOpsOrLatency(tiwsbt tiwsbt.Object) {
	s.display( tiwsbt)
}

func (s *ScreenDisplay) DisplayStages(essgben ps_table.Tabler) {
	s.display(essgben)
}

func (s *ScreenDisplay) DisplayUsers(users processlist.Object) {
	s.display(users)
}

func (s *ScreenDisplay) rel_time(last time.Time) float64 {
	now := time.Now()

	d := now.Sub(last)
	return d.Seconds()
}
