package widget

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestNoSlogInRenderPath ensures that render-path source files don't contain
// slog calls. Any slog call in a Render/View method creates a feedback loop:
// View → slog → console channel → Update → View → slog → ...
// This caused the CPU spin-up bug and keyboard starvation.
func TestNoSlogInRenderPath(t *testing.T) {
	// Files that are called from View/Render and must never use slog.
	renderFiles := []string{
		"window.go",
		"control.go",
		"label.go",
		"textview.go",
		"textinput.go",
		"button.go",
		"checkbox.go",
		"divider.go",
		"panel.go",
		"table.go",
		"teampanel.go",
		"container.go",
		"gameview.go",
		"overlay.go",
		"menu.go",
	}
	slogCall := regexp.MustCompile(`\bslog\.(Debug|Info|Warn|Error)\b`)

	for _, name := range renderFiles {
		// Files are in the same directory as this test.
		f, err := os.Open(name)
		if err != nil {
			// File might not exist in some build configurations; skip.
			continue
		}
		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			// Skip comments.
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			if slogCall.MatchString(line) {
				t.Errorf("%s:%d: slog call in render path (causes feedback loop): %s", name, lineNum, trimmed)
			}
		}
		f.Close()
	}
}
