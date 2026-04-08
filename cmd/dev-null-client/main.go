// dev-null-client is a graphical SSH client for dev-null servers.
//
// It connects via standard SSH but additionally supports charmap-based
// sprite rendering: games that declare a charmap have their PUA codepoints
// rendered as sprites from a sprite sheet instead of terminal glyphs.
//
// Use --no-gui for terminal mode: local game rendering output as ANSI to
// the current terminal, no graphical window. This gives a retro terminal vibe
// while still running game logic client-side for low latency.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/charmbracelet/colorprofile"
	xterm "github.com/charmbracelet/x/term"
	"github.com/hajimehoshi/ebiten/v2"

	"dev-null/internal/client"
	"dev-null/internal/datadir"
	"dev-null/internal/engine"
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
	localMode := flag.Bool("local", false, "start a headless server and connect to it")
	noSSH := flag.Bool("no-ssh", false, "skip SSH transport; connect chrome directly (requires --local, for testing)")
	address := flag.String("address", ":23234", "SSH listen address (local mode)")
	dataDir := flag.String("data-dir", datadir.DefaultDataDir(), "data directory containing games/ (local mode)")
	gameName := flag.String("game", "", "game to preload (local mode)")
	resumeName := flag.String("resume", "", "game/save to resume, e.g. orbits/autosave (local mode)")
	tickInterval := flag.Duration("tick-interval", 100*time.Millisecond, "server tick interval (local mode)")
	password := flag.String("password", "", "admin password (authenticates as admin on connect)")
	termFlag := flag.String("term", "", "force terminal color profile: truecolor, 256color, ansi, ascii")
	flag.Parse()

	if *noSSH && !*localMode {
		fmt.Fprintf(os.Stderr, "--no-ssh requires --local\n")
		os.Exit(1)
	}

	// Bootstrap bundled assets for local mode.
	if *localMode && *dataDir == datadir.DefaultDataDir() {
		if err := datadir.Bootstrap(datadir.InstallDir(), *dataDir, buildCommit); err != nil {
			fmt.Fprintf(os.Stderr, "bootstrap error: %v\n", err)
			os.Exit(1)
		}
	}

	// --local --no-ssh: direct transport, no SSH session.
	if *localMode && *noSSH {
		runDirect(*address, *dataDir, *player, *tickInterval, *gameName, *resumeName, *termFlag, *noGUI, *password)
		return
	}

	// --- All SSH paths: local and non-local, GUI and TUI ---
	// Establish the SSH connection, then run the appropriate renderer.
	// For --local, start a headless server first (in a background goroutine).

	var (
		conn    *client.SSHConn
		cleanup func()
	)

	if *localMode {
		var err error
		conn, cleanup, err = dialLocal(*address, *dataDir, *player, *port, *tickInterval, *gameName, *resumeName, *termFlag, *noGUI, *password)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("Connecting to %s:%d as %s...\n", *host, *port, *player)
		ptyW, ptyH := 0, 0
		if *noGUI {
			ptyW, ptyH, _ = xterm.GetSize(os.Stdin.Fd())
		}
		var err error
		conn, err = client.Dial(*host, *port, *player, *noGUI, *termFlag, *password, ptyW, ptyH, nil)
		if err != nil {
			log.Fatalf("Failed to connect: %v", err)
		}
		cleanup = func() {}
	}
	defer conn.Close()
	defer cleanup()

	if *noGUI {
		profile := detectClientProfile(*termFlag)
		if err := client.RunTerminal(conn, *player, profile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// GUI: run Ebitengine renderer. This call MUST happen directly in main()
	// on the main goroutine — Ebitengine locks OS thread 0 in init() and
	// creates the window from the first RunGame call on that thread.
	fmt.Println("Connected. Starting renderer...")

	fontFace := client.DefaultFontFace()
	dd := *dataDir
	if !*localMode {
		dd = datadir.DefaultDataDir()
	}
	fmt.Fprintf(os.Stderr, "DEBUG main: creating game\n")
	game := client.NewGame(conn, fontFace, 1200, 800, *player, dd)

	ebiten.SetWindowSize(1200, 800)
	if *localMode {
		ebiten.SetWindowTitle("dev-null (local)")
	} else {
		ebiten.SetWindowTitle("dev-null")
	}
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	fmt.Fprintf(os.Stderr, "DEBUG main: ebiten.RunGame starting\n")
	if err := ebiten.RunGameWithOptions(game, &ebiten.RunGameOptions{
		InitUnfocused: false,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "RunGame returned error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "DEBUG main: ebiten.RunGame returned cleanly\n")
}

// dialLocal starts a headless server in a background goroutine and dials it.
// Returns the SSH connection and a cleanup function that shuts down the server.
func dialLocal(address, dataDir, playerName string, port int, tickInterval time.Duration, gameName, resumeName, termFlag string, noGUI bool, password string) (*client.SSHConn, func(), error) {
	sshPort := port
	if idx := strings.LastIndex(address, ":"); idx >= 0 {
		if p := address[idx+1:]; p != "" {
			fmt.Sscanf(p, "%d", &sshPort)
		}
	}

	// Convert --game/--resume to init commands sent over the SSH session.
	var initCmds []string
	if resumeName != "" {
		initCmds = append(initCmds, "/game-resume "+resumeName)
	} else if gameName != "" {
		initCmds = append(initCmds, "/game-load "+gameName)
	}

	// Start headless server entirely in a background goroutine.
	serverCtx, serverCancel := context.WithCancel(context.Background())

	ready := make(chan struct{})
	serverErr := make(chan error, 1)
	go func() {
		app, err := server.New(address, password, dataDir, tickInterval)
		if err != nil {
			serverErr <- err
			return
		}
		app.SetLocalPlayerName(playerName)
		app.SetShutdownFunc(serverCancel)
		serverErr <- app.StartWithReady(serverCtx, ready)
	}()

	select {
	case <-ready:
	case err := <-serverErr:
		serverCancel()
		return nil, nil, fmt.Errorf("server failed to start: %w", err)
	}

	// Connect as a normal client.
	ptyW, ptyH := 0, 0
	if noGUI {
		ptyW, ptyH, _ = xterm.GetSize(os.Stdout.Fd())
	}
	conn, err := client.Dial("127.0.0.1", sshPort, playerName, noGUI, termFlag, password, ptyW, ptyH, initCmds)
	if err != nil {
		serverCancel()
		return nil, nil, fmt.Errorf("local SSH dial: %w", err)
	}

	return conn, serverCancel, nil
}

// runDirect runs the --no-ssh path: server + chrome connected directly,
// no SSH transport. Useful for isolating rendering issues from transport.
func runDirect(address, dataDir, playerName string, tickInterval time.Duration, gameName, resumeName, termFlag string, noGUI bool, password string) {
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
