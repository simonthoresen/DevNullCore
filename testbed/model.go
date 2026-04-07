package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

var flagFrames = flag.Int("frames", 0, "quit after N frames (0 = run until q/ctrl+c)")

// ANSI 256-color codes for the 15 static rows.
var rowColors = [15]int{
	196, 202, 208, 214, 220,
	118, 46, 47, 48, 49,
	51, 45, 21, 57, 201,
}

// All non-ASCII chars below are EAW=N (Neutral) or EAW=Na (Narrow) so the
// delta renderer's column arithmetic matches what the terminal actually renders.
// EAW=A (Ambiguous) chars like box-drawing, block elements, Greek letters,
// em-dash, and geometric shapes must NOT be used: bubbletea counts them as
// 1-column wide, but terminals in some locales render them as 2-column, causing
// cursor-left moves to land at wrong positions and staircase corruption.

// horzRunes: marquee text. ⟹⟸ are U+27F9/U+27F8 (Supplemental Arrows-A, EAW=N).
var horzRunes = []rune("  ⟹ SSH DELTA RENDER TEST ⟸  | colors | leading-spaces | scroll |  ")

// vertRunes: vertical scroll column. Mathematical operators U+2295-U+22A1, EAW=N.
// Braille patterns U+2800-U+28FF are also EAW=N and give a "loading bar" look.
// All EAW=N verified: ⊗(U+2297) ⊚(U+229A) ⊛(U+229B) ⊞(U+229E) ⊡(U+22A1)
// ⊠(U+22A0) ⊟(U+229F) ⊝(U+229D) ⊜(U+229C). Excluded: ⊕(U+2295 EAW=A) ⊙(U+2299 EAW=A).
var vertRunes = []rune("⊗⊚⊛⊞⊡⊠⊟⊝⊜⊛⊚⊗⊚⊜⊗⊟" +
	"⠁⠂⠄⠈⠐⠠⡀⢀⣀⣠⣤⣦⣶⣿⣾⣶⣤⣠⣀⢀⡀⠠⠐⠈⠄⠂⠁")

// diagRunes: diagonal element. Plain backslash keeps it unambiguous.
// ⊡⊞⊟⊠ (U+22A1/22A0/229F/229E, EAW=N) add visual interest without width issues.
var diagRunes = []rune(`\⊡\⊞\⊟\⊠\⊡\⊞`)

const (
	horzWidth    = 56
	diagWidth    = 20
	leftColWidth = 33 // visible columns: indent(i) + "[Row NN] " (9) + letters(22-i) = 31, +2 gap
)

type model struct{ frame int }

func (m model) Init() tea.Cmd { return tick() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case time.Time:
		m.frame++
		if *flagFrames > 0 && m.frame >= *flagFrames {
			return m, tea.Quit
		}
		return m, tick()
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders three scroll animations with EAW=N-only non-ASCII chars so the
// delta renderer's column arithmetic stays accurate across all frames.
func (m model) View() tea.View {
	var b strings.Builder

	// Header: only the frame number changes each tick.
	fmt.Fprintf(&b, "\x1b[1mSSH delta render test - frame %04d\x1b[0m\n\n", m.frame)

	// Horizontal marquee: scrolls left one char per frame.
	hLen := len(horzRunes)
	off := m.frame % hLen
	doubled := append(horzRunes, horzRunes...) //nolint:gocritic
	window := string(doubled[off : off+horzWidth])
	fmt.Fprintf(&b, "\x1b[1;38;5;226m[ %s ]\x1b[0m\n\n", window)

	vLen := len(vertRunes)
	dLen := len(diagRunes)

	for i := 0; i < 15; i++ {
		// Left cell: correctly padded to leftColWidth VISIBLE columns.
		// Padding uses the plain-text width, not the byte-length of the
		// ANSI-escaped string (which would include escape byte counts).
		content := fmt.Sprintf("[Row %02d] %s", i, strings.Repeat(string(rune('A'+i%26)), 22-i))
		indent := strings.Repeat(" ", i)
		visibleWidth := i + len(content) // all ASCII: rune count == column count
		pad := leftColWidth - visibleWidth
		if pad < 0 {
			pad = 0
		}
		leftCell := indent +
			fmt.Sprintf("\x1b[38;5;%dm%s\x1b[0m", rowColors[i], content) +
			strings.Repeat(" ", pad)

		// Vertical scroll: chars rise upward (newest at row 0).
		vIdx := ((m.frame - i) % vLen + vLen) % vLen

		// Diagonal stripe: one char at (frame+i) % diagWidth; rest are spaces.
		diagPos := (m.frame + i) % diagWidth
		diagSection := []rune(strings.Repeat(" ", diagWidth))
		diagSection[diagPos] = diagRunes[(m.frame+i)%dLen]

		fmt.Fprintf(&b, "%s| \x1b[38;5;51m%c\x1b[0m |\x1b[38;5;201m%s\x1b[0m\n",
			leftCell, vertRunes[vIdx], string(diagSection))
	}

	fmt.Fprintf(&b, "\n(q=quit)\n")
	return tea.NewView(b.String())
}

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return t })
}
