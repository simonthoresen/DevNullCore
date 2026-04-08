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
	"os/exec"
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

	// --- All SSH paths (local and remote, GUI and TUI) ---
	// Sequential: 1) start server (if local), 2) connect SSH, 3) run renderer.

	var (
		conn          *client.SSHConn
		serverCleanup = func() {}
	)

	if *localMode {
		var err error
		conn, serverCleanup, err = dialLocal(*address, *dataDir, *player, *port, *tickInterval, *gameName, *resumeName, *termFlag, *noGUI, *password)
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
	}
	defer conn.Close()
	defer serverCleanup()

	// TUI: render in terminal.
	if *noGUI {
		profile := detectClientProfile(*termFlag)
		if err := client.RunTerminal(conn, *player, profile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// GUI: render in Ebitengine window.
	fmt.Println("Connected. Starting renderer...")
	title := "dev-null"
	if *localMode {
		title = "dev-null (local)"
	}
	dd := *dataDir
	if !*localMode {
		dd = datadir.DefaultDataDir()
	}
	renderer := client.NewClientRenderer(conn, 1200, 800, *player, dd)
	if err := display.RunWindow(renderer, title, 1200, 800); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// dialLocal starts a headless server as a subprocess and connects to it.
// Returns the connection and a cleanup function that kills the subprocess.
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

	// Find the server binary next to the client binary.
	exe, _ := os.Executable()
	serverBin := filepath.Join(filepath.Dir(exe), "dev-null-server.exe")
	if _, err := os.Stat(serverBin); err != nil {
		// Fallback: try in data dir.
		serverBin = filepath.Join(dataDir, "dev-null-server.exe")
	}

	// Start headless server as a subprocess. Running it in-process prevents
	// Ebitengine from creating its window on Windows (Bubble Tea's per-session
	// console handling interferes with the Win32 message loop).
	args := []string{"--no-gui", "--data-dir", dataDir, "--address", address}
	if password != "" {
		args = append(args, "--password", password)
	}
	cmd := exec.Command(serverBin, args...)
	cmd.Stdout = os.Stderr // show server logs
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("start server: %w (looked for %s)", err, serverBin)
	}

	cleanup := func() {
		cmd.Process.Signal(os.Interrupt)
		done := make(chan struct{})
		go func() { cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			cmd.Process.Kill()
		}
	}

	// Poll until the server is listening.
	var conn *client.SSHConn
	ptyW, ptyH := 0, 0
	if noGUI {
		ptyW, ptyH, _ = xterm.GetSize(os.Stdout.Fd())
	}
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		c, err := client.Dial("127.0.0.1", sshPort, playerName, noGUI, termFlag, password, ptyW, ptyH, initCmds)
		if err == nil {
			conn = c
			break
		}
	}
	if conn == nil {
		cleanup()
		return nil, nil, fmt.Errorf("could not connect to local server on port %d", sshPort)
	}

	return conn, cleanup, nil
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
