package main

import (
	"image/color"
	"testing"

	"dev-null/internal/client"
)

func TestLauncherSceneRendersVisiblePixels(t *testing.T) {
	lr := client.NewLocalRenderer()
	lr.LoadGame([]client.GameSrcFile{{Name: "launcher_scene.js", Content: launcherSceneScript}})
	lr.SetState([]byte(`{"_gameTime":1.5}`))

	img := lr.RenderCanvasImage("launcher", 320, 200)
	if img == nil {
		t.Fatal("expected launcher scene image, got nil")
	}

	bg := color.RGBA{R: 0x03, G: 0x05, B: 0x0d, A: 0xff}
	nonBackground := 0
	brightPixels := 0

	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
			if c != bg {
				nonBackground++
			}
			if c.R > 180 || c.G > 180 || c.B > 180 {
				brightPixels++
			}
		}
	}

	if nonBackground < 500 {
		t.Fatalf("expected scene details to draw, non-background pixel count=%d", nonBackground)
	}
	if brightPixels < 40 {
		t.Fatalf("expected stars/highlights to render, bright pixel count=%d", brightPixels)
	}
}
