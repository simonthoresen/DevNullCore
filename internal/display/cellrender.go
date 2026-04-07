// Package display provides pluggable display backends (terminal TUI vs Ebitengine GUI)
// for rendering ImageBuffer-based UI to different output targets.
package display

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/bitmapfont/v4"

	"dev-null/internal/render"
)

// CellW and CellH are the pixel dimensions of a single terminal cell.
const (
	CellW = 10
	CellH = 20
)

// DefaultFontFace returns the built-in bitmap font face for terminal rendering.
func DefaultFontFace() text.Face {
	return text.NewGoXFace(bitmapfont.Face)
}

// sharedPixel is a 1x1 white image reused for all background fills.
// Colored via ColorScale to avoid per-cell image allocation.
var sharedPixel *ebiten.Image

func init() {
	sharedPixel = ebiten.NewImage(1, 1)
	sharedPixel.Fill(color.White)
}

// DrawImageBuffer renders an ImageBuffer to an Ebitengine screen image.
// Each cell is drawn as a colored background rectangle, then foreground text.
func DrawImageBuffer(screen *ebiten.Image, buf *render.ImageBuffer, fontFace text.Face) {
	for cy := 0; cy < buf.Height; cy++ {
		for cx := 0; cx < buf.Width; cx++ {
			p := &buf.Pixels[cy*buf.Width+cx]
			px := cx * CellW
			py := cy * CellH

			// Background: draw a scaled 1x1 white pixel with color tint.
			if p.Bg != nil {
				r, g, b, _ := p.Bg.RGBA()
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(CellW, CellH)
				op.GeoM.Translate(float64(px), float64(py))
				op.ColorScale.ScaleWithColor(color.RGBA{
					R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 255,
				})
				screen.DrawImage(sharedPixel, op)
			}

			// Foreground text.
			if p.Char != ' ' && p.Char != 0 {
				fg := color.RGBA{R: 204, G: 204, B: 204, A: 255}
				if p.Fg != nil {
					r, g, b, _ := p.Fg.RGBA()
					fg = color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 255}
				}
				dop := &text.DrawOptions{}
				dop.GeoM.Translate(float64(px), float64(py))
				dop.ColorScale.ScaleWithColor(fg)
				text.Draw(screen, string(p.Char), fontFace, dop)
			}
		}
	}
}
