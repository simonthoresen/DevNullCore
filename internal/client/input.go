package client

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// handleInput maps Ebitengine key events to SSH-compatible escape sequences.
func (r *ClientRenderer) handleInput() {
	alt := ebiten.IsKeyPressed(ebiten.KeyAlt)
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl)

	// Character input (typed text) — skip when Alt or Ctrl is held.
	if !alt && !ctrl {
		runes := ebiten.AppendInputChars(nil)
		for _, ch := range runes {
			r.conn.Write([]byte(string(ch)))
		}
	}

	// Special keys.
	for _, key := range specialKeys {
		if inpututil.IsKeyJustPressed(key.ekey) {
			r.conn.Write([]byte(key.seq))
		}
	}

	// Alt+letter for menu shortcuts (e.g. Alt+F → ESC f).
	if alt && !ctrl {
		for key := ebiten.KeyA; key <= ebiten.KeyZ; key++ {
			if inpututil.IsKeyJustPressed(key) {
				letter := byte('a' + (key - ebiten.KeyA))
				r.conn.Write([]byte{0x1b, letter})
			}
		}
	}

	// Ctrl+letter combos (Ctrl+A = 0x01, Ctrl+B = 0x02, ..., Ctrl+Z = 0x1A).
	if ctrl && !alt {
		for key := ebiten.KeyA; key <= ebiten.KeyZ; key++ {
			if inpututil.IsKeyJustPressed(key) {
				r.conn.Write([]byte{byte(1 + (key - ebiten.KeyA))})
			}
		}
	}

	// Mouse events — send as SGR (mode 1006) escape sequences.
	r.handleMouseInput()
}

// handleMouseInput sends mouse click and scroll events as SGR escape sequences.
func (r *ClientRenderer) handleMouseInput() {
	cx, cy := ebiten.CursorPosition()
	cellX := cx / cellW()
	cellY := cy / cellH()

	// Left click.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		r.conn.Write([]byte(fmt.Sprintf("\x1b[<%d;%d;%dM", 0, cellX+1, cellY+1)))
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		r.conn.Write([]byte(fmt.Sprintf("\x1b[<%d;%d;%dm", 0, cellX+1, cellY+1)))
	}

	// Right click.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		r.conn.Write([]byte(fmt.Sprintf("\x1b[<%d;%d;%dM", 2, cellX+1, cellY+1)))
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight) {
		r.conn.Write([]byte(fmt.Sprintf("\x1b[<%d;%d;%dm", 2, cellX+1, cellY+1)))
	}

	// Middle click.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle) {
		r.conn.Write([]byte(fmt.Sprintf("\x1b[<%d;%d;%dM", 1, cellX+1, cellY+1)))
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonMiddle) {
		r.conn.Write([]byte(fmt.Sprintf("\x1b[<%d;%d;%dm", 1, cellX+1, cellY+1)))
	}

	// Scroll wheel.
	_, scrollY := ebiten.Wheel()
	if scrollY > 0 {
		r.conn.Write([]byte(fmt.Sprintf("\x1b[<%d;%d;%dM", 64, cellX+1, cellY+1)))
	} else if scrollY < 0 {
		r.conn.Write([]byte(fmt.Sprintf("\x1b[<%d;%d;%dM", 65, cellX+1, cellY+1)))
	}
}

type keyMapping struct {
	ekey ebiten.Key
	seq  string
}

var specialKeys = []keyMapping{
	{ebiten.KeyEnter, "\r"},
	{ebiten.KeyBackspace, "\x7f"},
	{ebiten.KeyTab, "\t"},
	{ebiten.KeyEscape, "\x1b"},
	{ebiten.KeyUp, "\x1b[A"},
	{ebiten.KeyDown, "\x1b[B"},
	{ebiten.KeyRight, "\x1b[C"},
	{ebiten.KeyLeft, "\x1b[D"},
	{ebiten.KeyHome, "\x1b[H"},
	{ebiten.KeyEnd, "\x1b[F"},
	{ebiten.KeyPageUp, "\x1b[5~"},
	{ebiten.KeyPageDown, "\x1b[6~"},
	{ebiten.KeyDelete, "\x1b[3~"},
	{ebiten.KeyF1, "\x1bOP"},
	{ebiten.KeyF2, "\x1bOQ"},
	{ebiten.KeyF3, "\x1bOR"},
	{ebiten.KeyF4, "\x1bOS"},
	{ebiten.KeyF5, "\x1b[15~"},
	{ebiten.KeyF6, "\x1b[17~"},
	{ebiten.KeyF7, "\x1b[18~"},
	{ebiten.KeyF8, "\x1b[19~"},
	{ebiten.KeyF9, "\x1b[20~"},
	{ebiten.KeyF10, "\x1b[21~"},
	{ebiten.KeyF11, "\x1b[23~"},
	{ebiten.KeyF12, "\x1b[24~"},
}
