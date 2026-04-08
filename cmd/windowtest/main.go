// Minimal test to isolate Ebitengine window creation with server goroutine.
package main

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"dev-null/internal/server"
)

type testGame struct{ frame int }

func (g *testGame) Update() error {
	g.frame++
	if g.frame == 1 {
		fmt.Fprintf(os.Stderr, "TEST: first Update()\n")
	}
	if g.frame > 300 { // ~5 seconds at 60fps
		return ebiten.Termination
	}
	return nil
}

func (g *testGame) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 40, G: 80, B: 160, A: 255})
	ebitenutil.DebugPrint(screen, fmt.Sprintf("Frame: %d", g.frame))
}

func (g *testGame) Layout(w, h int) (int, int) { return w, h }

func main() {
	mode := "bare"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	fmt.Fprintf(os.Stderr, "TEST MODE: %s\n", mode)

	switch mode {
	case "bare":
		// Just Ebitengine, nothing else.
	case "server-only":
		// Start server goroutine but don't connect.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ready := make(chan struct{})
		go func() {
			app, _ := server.New(":23235", "", ".", 100*time.Millisecond)
			app.StartWithReady(ctx, ready)
		}()
		<-ready
		fmt.Fprintf(os.Stderr, "TEST: server ready\n")
	case "server-and-ssh":
		// Start server AND connect SSH.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ready := make(chan struct{})
		go func() {
			app, _ := server.New(":23235", "", ".", 100*time.Millisecond)
			app.SetLocalPlayerName("test")
			app.StartWithReady(ctx, ready)
		}()
		<-ready
		fmt.Fprintf(os.Stderr, "TEST: server ready, dialing SSH\n")
		// Import client to dial
		import_ssh(ctx)
	}

	ebiten.SetWindowSize(400, 300)
	ebiten.SetWindowTitle(fmt.Sprintf("TEST: %s", mode))
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	fmt.Fprintf(os.Stderr, "TEST: calling RunGame\n")
	if err := ebiten.RunGame(&testGame{}); err != nil {
		fmt.Fprintf(os.Stderr, "TEST: RunGame error: %v\n", err)
	}
	fmt.Fprintf(os.Stderr, "TEST: RunGame returned\n")
}

func import_ssh(ctx context.Context) {
	// Placeholder — we'll test server-only first.
}
