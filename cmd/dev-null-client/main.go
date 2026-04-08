// dev-null-client is a graphical SSH client for dev-null servers.
//
// It connects via standard SSH but additionally supports charmap-based
// sprite rendering: games that declare a charmap have their PUA codepoints
// rendered as sprites from a sprite sheet instead of terminal glyphs.
//
// --local mode runs a server in-process with no SSH transport — the chrome
// model is connected directly to the display backend.
//
// Use --no-gui for terminal mode: renders as ANSI to the current terminal.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/colorprofile"
	xterm "github.com/charmbracelet/x/term"

	"dev-null/internal/client"
	"dev-null/internal/datadir"
	"dev-null/internal/display"
	"dev-null/internal/engine"
	"dev-null/internal/runlog"
	"dev-null/internal/server"
)

// buildCommit, buildDate, and buildRemote are injected at build time via -ldflags.
var buildCommit = "dev"
var buildDate = "unknown"
var buildRemote = ""

func main() {
	fmt.Printf("dev-null-client %s (%s)\n", buildCommit, buildDate)
	engine.SetBuildInfo(buildDate, buildRemote)
	host := flag.String("host", "localhost", "server hostname")
	port := flag.Int("port", 23234, "server SSH port")
	player := flag.String("player", defaultPlayer(), "player name")
	noGUI := flag.Bool("no-gui", false, "run in terminal mode (TUI) instead of opening a graphical window")
	localMode := flag.Bool("local", false, "run a local server in-process (no SSH, no network)")
	address := flag.String("address", ":23234", "SSH listen address (local mode)")
	dataDir := flag.String("data-dir", datadir.DefaultDataDir(), "data directory containing games/")
	gameName := flag.String("game", "", "game to load (local mode)")
	resumeName := flag.String("resume", "", "game/save to resume, e.g. orbits/autosave (local mode)")
	tickInterval := flag.Duration("tick-interval", 100*time.Millisecond, "server tick interval (local mode)")
	password := flag.String("password", "", "admin password (authenticates as admin on connect)")
	termFlag := flag.String("term", "", "force terminal color profile: truecolor, 256color, ansi, ascii")
	flag.Parse()

	// Set up logging to data-dir/logs/client-<timestamp>.log.
	logsDir := filepath.Join(*dataDir, "logs")
	cleanupLog, err := runlog.ConfigureAuto(logsDir, "client")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not configure logging: %v\n", err)
		os.Exit(1)
	}
	defer cleanupLog() //nolint:errcheck

	// Bootstrap bundled assets for local mode.
	if *localMode && *dataDir == datadir.DefaultDataDir() {
		if err := datadir.Bootstrap(datadir.InstallDir(), *dataDir, buildCommit); err != nil {
			fmt.Fprintf(os.Stderr, "bootstrap error: %v\n", err)
			os.Exit(1)
		}
	}

	// --local: run server in-process, no SSH. Chrome model connects directly.
	if *localMode {
		runLocal(*address, *dataDir, *player, *tickInterval, *gameName, *resumeName, *termFlag, *noGUI, *password)
		return
	}

	// --- Remote: connect to a server via SSH ---

	fmt.Printf("Connecting to %s:%d as %s...\n", *host, *port, *player)
	ptyW, ptyH := 0, 0
	if *noGUI {
		ptyW, ptyH, _ = xterm.GetSize(os.Stdin.Fd())
	}
	conn, err := client.Dial(*host, *port, *player, *noGUI, *termFlag, *password, ptyW, ptyH, nil)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	if *noGUI {
		profile := detectClientProfile(*termFlag)
		if err := client.RunTerminal(conn, *player, profile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Println("Connected. Starting renderer...")
	renderer := client.NewClientRenderer(conn, 1200, 800, *player, datadir.DefaultDataDir())
	if err := display.RunWindow(renderer, "dev-null", 1200, 800); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runLocal runs a server in-process with the chrome model connected directly
// to the display backend. No SSH, no network, no subprocess.
func runLocal(address, dataDir, playerName string, tickInterval time.Duration, gameName, resumeName, termFlag string, noGUI bool, password string) {
	app, err := server.New(address, password, dataDir, tickInterval)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating server: %v\n", err)
		os.Exit(1)
	}
	if resumeName != "" {
		parts := strings.SplitN(resumeName, "/", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "--resume requires game/save format, e.g. orbits/autosave\n")
			os.Exit(1)
		}
		if err := app.PreloadResume(parts[0], parts[1]); err != nil {
			fmt.Fprintf(os.Stderr, "resume %s: %v\n", resumeName, err)
			os.Exit(1)
		}
	} else if gameName != "" {
		if err := app.PreloadGame(gameName); err != nil {
			fmt.Fprintf(os.Stderr, "load game %s: %v\n", gameName, err)
			os.Exit(1)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := app.RunDirect(ctx, playerName, termFlag, noGUI); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// detectClientProfile returns the color profile for client-side terminal rendering.
func detectClientProfile(termFlag string) colorprofile.Profile {
	if termFlag != "" {
		switch strings.ToLower(termFlag) {
		case "truecolor", "24bit":
			return colorprofile.TrueColor
		case "256color", "256":
			return colorprofile.ANSI256
		case "ansi", "16color", "16":
			return colorprofile.ANSI
		case "ascii", "none", "no-color":
			return colorprofile.ASCII
		default:
			fmt.Fprintf(os.Stderr, "unknown --term value %q (valid: truecolor, 256color, ansi, ascii)\n", termFlag)
		}
	}
	return colorprofile.Detect(os.Stderr, os.Environ())
}

func defaultPlayer() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return "player"
}
