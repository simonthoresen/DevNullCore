package rendertest

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/parser"
)

// vt100 is a minimal terminal emulator that reconstructs the current screen
// state from a raw byte stream containing ANSI/VT100 escape sequences.
//
// It handles the subset emitted by bubbletea v2's cursed renderer:
//   - CSI H / f  — cursor position (CUP/HVP)
//   - CSI A/B/C/D — relative cursor movement
//   - CSI G / d  — column / line absolute
//   - CSI J      — erase display
//   - CSI K      — erase line
//   - C0 \r \n \b — carriage return, line feed, backspace
//   - SGR (m), private-mode h/l — silently ignored
//   - OSC/DCS/APC strings — silently ignored
type vt100 struct {
	rows, cols int
	cells      [][]rune
	row, col   int // cursor position, 0-indexed
}

func newVT100(rows, cols int) *vt100 {
	cells := make([][]rune, rows)
	for i := range cells {
		cells[i] = make([]rune, cols)
		for j := range cells[i] {
			cells[i][j] = ' '
		}
	}
	return &vt100{rows: rows, cols: cols, cells: cells}
}

// feed processes all bytes in data through the emulator, updating screen state.
func (t *vt100) feed(data []byte) {
	p := ansi.NewParser()
	for _, b := range data {
		prevState := p.State()
		action := p.Advance(b)
		switch action {
		case parser.PrintAction:
			t.put(p.Rune())
		case parser.ExecuteAction:
			t.execute(p.Control())
		case parser.DispatchAction:
			switch prevState {
			case parser.CsiEntryState, parser.CsiParamState, parser.CsiIntermediateState:
				t.handleCSI(p)
			// ESC, OSC, DCS, APC — all ignored
			}
		}
	}
}

func (t *vt100) put(r rune) {
	if r == 0 {
		return
	}
	if t.col >= t.cols {
		t.col = 0
		t.row++
	}
	if t.row < 0 || t.row >= t.rows {
		return
	}
	t.cells[t.row][t.col] = r
	t.col++
}

func (t *vt100) execute(b byte) {
	switch b {
	case '\r':
		t.col = 0
	case '\n':
		t.row++
		if t.row >= t.rows {
			t.row = t.rows - 1
		}
	case '\b':
		if t.col > 0 {
			t.col--
		}
	case '\t':
		// advance to next 8-column tab stop, clamped to last column
		next := (t.col/8 + 1) * 8
		if next >= t.cols {
			next = t.cols - 1
		}
		t.col = next
	}
}

func (t *vt100) handleCSI(p *ansi.Parser) {
	final := rune(p.Command() & 0xFF)
	switch final {
	case 'H', 'f': // cursor position / HVP — \x1b[row;colH (1-based)
		row, _ := p.Param(0, 1)
		col, _ := p.Param(1, 1)
		t.row = clampInt(row-1, 0, t.rows-1)
		t.col = clampInt(col-1, 0, t.cols-1)
	case 'A': // cursor up
		n, _ := p.Param(0, 1)
		t.row = maxInt(0, t.row-n)
	case 'B': // cursor down
		n, _ := p.Param(0, 1)
		t.row = minInt(t.rows-1, t.row+n)
	case 'C': // cursor forward (right)
		n, _ := p.Param(0, 1)
		t.col = minInt(t.cols-1, t.col+n)
	case 'D': // cursor back (left)
		n, _ := p.Param(0, 1)
		t.col = maxInt(0, t.col-n)
	case 'G': // cursor horizontal absolute (column, 1-based)
		col, _ := p.Param(0, 1)
		t.col = clampInt(col-1, 0, t.cols-1)
	case 'd': // line position absolute (row, 1-based)
		row, _ := p.Param(0, 1)
		t.row = clampInt(row-1, 0, t.rows-1)
	case 'J': // erase in display
		n, _ := p.Param(0, 0)
		t.eraseDisplay(n)
	case 'K': // erase in line
		n, _ := p.Param(0, 0)
		t.eraseLine(n)
	case 'X': // erase character (ECH)
		n, _ := p.Param(0, 1)
		for i := t.col; i < t.col+n && i < t.cols; i++ {
			if t.row >= 0 && t.row < t.rows {
				t.cells[t.row][i] = ' '
			}
		}
	case '@': // insert character (ICH) — shift right
		n, _ := p.Param(0, 1)
		if t.row >= 0 && t.row < t.rows {
			row := t.cells[t.row]
			copy(row[t.col+n:], row[t.col:])
			for i := t.col; i < t.col+n && i < t.cols; i++ {
				row[i] = ' '
			}
		}
	case 'P': // delete character (DCH) — shift left
		n, _ := p.Param(0, 1)
		if t.row >= 0 && t.row < t.rows {
			row := t.cells[t.row]
			copy(row[t.col:], row[t.col+n:])
			for i := t.cols - n; i < t.cols; i++ {
				row[i] = ' '
			}
		}
	// 'm' (SGR), 'h'/'l' (DEC modes), 'r' (margins), 's'/'u' (save/restore),
	// 'n' (device status), 'c' (device attributes) — all silently ignored.
	}
}

func (t *vt100) eraseDisplay(n int) {
	switch n {
	case 0: // from cursor to end of screen
		t.eraseLinePart(t.row, t.col, t.cols)
		for r := t.row + 1; r < t.rows; r++ {
			t.eraseLinePart(r, 0, t.cols)
		}
	case 1: // from start of screen to cursor
		for r := 0; r < t.row; r++ {
			t.eraseLinePart(r, 0, t.cols)
		}
		t.eraseLinePart(t.row, 0, t.col+1)
	case 2, 3: // whole screen
		for r := 0; r < t.rows; r++ {
			t.eraseLinePart(r, 0, t.cols)
		}
	}
}

func (t *vt100) eraseLine(n int) {
	switch n {
	case 0: // from cursor to end of line
		t.eraseLinePart(t.row, t.col, t.cols)
	case 1: // from start of line to cursor
		t.eraseLinePart(t.row, 0, t.col+1)
	case 2: // whole line
		t.eraseLinePart(t.row, 0, t.cols)
	}
}

func (t *vt100) eraseLinePart(row, from, to int) {
	if row < 0 || row >= t.rows {
		return
	}
	for c := from; c < to && c < t.cols; c++ {
		t.cells[row][c] = ' '
	}
}

// String returns the screen content as newline-separated lines.
// Trailing spaces are stripped from each line, and trailing blank lines
// are dropped — matching the format of the unit-test golden files.
func (t *vt100) String() string {
	lines := make([]string, t.rows)
	for i, row := range t.cells {
		lines[i] = strings.TrimRight(string(row), " ")
	}
	// Drop trailing blank lines.
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
