package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dev-null/internal/server"
)

func main() {
	game    := flag.String("game",     "",       "game to preload")
	dataDir := flag.String("data-dir", "dist",   "directory containing games/, logs/")
	player  := flag.String("player",   "player", "player name")
	term    := flag.String("term",     "",       "color profile: truecolor, 256color, ansi, ascii")
	flag.Parse()

	app, err := server.New("", "", *dataDir, 100*time.Millisecond)
	if err != nil {
		fmt.Fprintf(os.Stderr, "server init: %v\n", err)
		os.Exit(1)
	}

	if *game != "" {
		if err := app.PreloadGame(*game); err != nil {
			fmt.Fprintf(os.Stderr, "load game %q: %v\n", *game, err)
			os.Exit(1)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.RunDirect(ctx, *player, *term); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
