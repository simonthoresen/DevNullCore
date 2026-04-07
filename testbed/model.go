package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

var flagFrames = flag.Int("frames", 0, "quit after N frames (0 = run until q/ctrl+c)")

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

// View renders 15 numbered rows of fixed-width content.
// Each row must start at column 0. If ONLCR is missing (bare \n, no \r),
// the cursor moves down without returning to col 0, and each subsequent row
// is shifted right — a staircase pattern that is immediately obvious.
func (m model) View() tea.View {
	var b strings.Builder
	for i := 0; i < 15; i++ {
		fmt.Fprintf(&b, "Row %02d: [%s]\n", i, strings.Repeat(string(rune('A'+i%26)), 40))
	}
	fmt.Fprintf(&b, "\nFrame: %d  (q=quit)\n", m.frame)
	return tea.NewView(b.String())
}

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return t })
}
