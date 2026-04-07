package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

var flagFrames = flag.Int("frames", 0, "quit after N frames (0 = run until q/ctrl+c)")

// ANSI 256-color foreground codes for the 15 rows.
var rowColors = [15]int{
	196, 202, 208, 214, 220, // reds → oranges → yellow
	118, 46, 47, 48, 49,     // greens
	51, 45, 21, 57, 201,     // cyans → blues → magentas
}

type model struct{ frame int }

func (m model) Init() tea.Cmd {
	return tick()
}

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

// View renders 15 rows, each with:
//   - leading spaces (0..14) to stress cursor-column tracking
//   - ANSI 256-color foreground to stress color state across delta updates
//   - fixed content so delta renderer only repaints the changed frame counter
func (m model) View() tea.View {
	var b strings.Builder

	// Bold + colored header — changes every frame so delta renders it each tick.
	fmt.Fprintf(&b, "\x1b[1mSSH delta render test — frame %04d\x1b[0m\n\n", m.frame)

	for i := 0; i < 15; i++ {
		indent := strings.Repeat(" ", i)
		color := rowColors[i]
		letter := string(rune('A' + i%26))
		content := strings.Repeat(letter, 40-i) // shrinks to compensate for indent
		// Row is static content; only the header above changes each frame.
		fmt.Fprintf(&b, "%s\x1b[38;5;%dm[Row %02d] %s\x1b[0m\n", indent, color, i, content)
	}

	fmt.Fprintf(&b, "\n(q=quit)\n")
	return tea.NewView(b.String())
}

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return t })
}
