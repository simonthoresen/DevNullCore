package server

import (
	"fmt"
	"strings"
	"testing"
)

func TestRenderQR(t *testing.T) {
	const invite = `$env:NS='ABCDEFGH';irm https://raw.githubusercontent.com/simonthoresen/null-space/main/join.ps1|iex`

	qr, err := renderQR(invite)
	if err != nil {
		t.Fatalf("renderQR: %v", err)
	}

	lines := strings.Split(strings.TrimRight(qr, "\n"), "\n")
	if len(lines) < 5 {
		t.Fatalf("expected at least 5 lines, got %d", len(lines))
	}

	// All lines must have equal length (square QR modules → equal char columns).
	width := len([]rune(lines[0]))
	for i, line := range lines {
		if got := len([]rune(line)); got != width {
			t.Errorf("line %d width %d, want %d", i, got, width)
		}
	}

	// Every rune must be one of the 16 quadrant characters (or space).
	valid := map[rune]bool{
		' ': true, '▗': true, '▖': true, '▄': true,
		'▝': true, '▐': true, '▞': true, '▟': true,
		'▘': true, '▚': true, '▌': true, '▙': true,
		'▀': true, '▜': true, '▛': true, '█': true,
	}
	for i, line := range lines {
		for _, r := range line {
			if !valid[r] {
				t.Errorf("line %d contains unexpected rune %q (U+%04X)", i, r, r)
			}
		}
	}

	// Print so `go test -v` shows the QR code.
	fmt.Printf("\n%s%s\n", qr, invite)
}
