package widget

import (
	tea "charm.land/bubbletea/v2"

	"dev-null/internal/render"
	"dev-null/internal/theme"
)

// Button is a clickable button: [ Label ].
type Button struct {
	Label    string
	Align    string // "left" (default), "center", "right"
	OnPress  func()
	Disabled func() bool // if non-nil and returns true, button is grayed and non-functional

	WantTab     bool
	WantBackTab bool
}

func (b *Button) isDisabled() bool { return b.Disabled != nil && b.Disabled() }

func (b *Button) Focusable() bool { return !b.isDisabled() }
func (b *Button) HandleClick(rx, ry int) {
	if b.isDisabled() {
		return
	}
	if b.OnPress != nil {
		b.OnPress()
	}
}
func (b *Button) MinSize() (int, int)   { return len(b.Label) + 4, 1 } // "[ " + label + " ]"
func (b *Button) TabWant() (bool, bool) { return b.WantTab, b.WantBackTab }

// WantsEnter always consumes Enter when focused — pressing Enter activates
// the button. The framework's focus-chat action never reaches a focused button.
func (b *Button) WantsEnter() bool { return !b.isDisabled() }
func (b *Button) Update(msg tea.Msg) {
	b.WantTab = false
	b.WantBackTab = false
	if km, ok := msg.(tea.KeyPressMsg); ok {
		switch km.String() {
		case "enter", " ":
			if !b.isDisabled() && b.OnPress != nil {
				b.OnPress()
			}
		case "tab":
			b.WantTab = true
		case "shift+tab":
			b.WantBackTab = true
		}
	}
}
func (b *Button) Render(buf *render.ImageBuffer, x, y, width, height int, focused bool, layer *theme.Layer) {
	fg := layer.Fg
	bg := layer.Bg
	attr := render.PixelAttr(render.AttrNone)
	switch {
	case b.isDisabled():
		fg = layer.DisabledFg
	case focused:
		fg = layer.HighlightFg
		bg = layer.HighlightBg
		attr = render.AttrBold
	}
	label := "[ " + b.Label + " ]"
	labelW := len(label)
	startX := x
	switch b.Align {
	case "center":
		startX = x + (width-labelW)/2
		if startX < x {
			startX = x
		}
	case "right":
		startX = x + width - labelW
		if startX < x {
			startX = x
		}
	}
	col := startX
	for _, r := range label {
		if col >= x+width {
			break
		}
		buf.SetChar(col, y, r, fg, bg, attr)
		col++
	}
}
