package widget

import (
	tea "charm.land/bubbletea/v2"

	"null-space/common"
	"null-space/internal/theme"
)

// StatusBar is a Control that renders a single-row status bar with
// left-aligned and right-aligned text.
type StatusBar struct {
	LeftText  string
	RightText string
}

func (s *StatusBar) Focusable() bool      { return false }
func (s *StatusBar) MinSize() (int, int)  { return 1, 1 }
func (s *StatusBar) Update(_ tea.Msg)     {}

// Render fills the bar row with background color and writes left/right text.
func (s *StatusBar) Render(buf *common.ImageBuffer, x, y, width, height int, _ bool, layer *theme.Layer) {
	if width <= 0 || height <= 0 {
		return
	}

	fg := layer.FgC()
	bg := layer.BgC()

	// Fill the entire bar.
	buf.Fill(x, y, width, 1, ' ', fg, bg, common.AttrNone)

	// Left-aligned text.
	if s.LeftText != "" {
		buf.WriteString(x, y, s.LeftText, fg, bg, common.AttrNone)
	}

	// Right-aligned text.
	if s.RightText != "" {
		rightX := x + width - len(s.RightText)
		if rightX > x+len(s.LeftText) {
			buf.WriteString(rightX, y, s.RightText, fg, bg, common.AttrNone)
		}
	}
}
