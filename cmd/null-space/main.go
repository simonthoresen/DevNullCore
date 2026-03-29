package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/ssh"

	"null-space/common"
	"null-space/games/towerdefense"
	"null-space/internal/runlog"
	"null-space/server"
)

func main() {
	cleanupLog, err := runlog.ConfigureFromEnv("server")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not configure logging: %v\n", err)
		os.Exit(1)
	}
	defer cleanupLog() //nolint:errcheck

	var gameName string
	var password string
	var address string

	flag.StringVar(&gameName, "game", "towerdefense", "game module to run")
	flag.StringVar(&password, "password", "", "admin password")
	flag.StringVar(&address, "address", ":23234", "listen address")
	flag.Parse()

	game, resolvedName, err := loadGame(gameName)
	if err != nil {
		slog.Error("failed to load game", "game", gameName, "error", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	slog.Info("loaded game", "requested_game", gameName, "resolved_game", resolvedName)

	app, err := server.New(address, resolvedName, game, password)
	if err != nil {
		slog.Error("could not create server", "address", address, "error", err)
		fmt.Fprintf(os.Stderr, "could not create server: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	app.SetShutdownFunc(stop)

	if pinggyStatusFile := os.Getenv("NULL_SPACE_PINGGY_STATUS_FILE"); pinggyStatusFile != "" {
		slog.Info("enabling pinggy log bridge", "status_file", pinggyStatusFile)
		app.EnablePinggyLogBridge(ctx, pinggyStatusFile)
	}

	if stdinInfo, err := os.Stdin.Stat(); err == nil && (stdinInfo.Mode()&os.ModeCharDevice) != 0 {
		slog.Info("enabling local console")
		app.EnableLocalConsole(ctx, stop, os.Stdin, os.Stdout)
	}

	slog.Info("starting server", "address", address)
	if err := app.Start(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		slog.Error("server failed", "error", err)
		fmt.Fprintf(os.Stderr, "server failed: %v\n", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}

func loadGame(name string) (common.Game, string, error) {
	switch name {
	case "towerdefense", "tower-defense", "td":
		return towerdefense.New(), "towerdefense", nil
	default:
		return nil, "", fmt.Errorf("unknown game %q", name)
	}
}
