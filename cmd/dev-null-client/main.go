// dev-null-client is a graphical SSH client for dev-null servers.
//
// It connects via standard SSH but additionally supports charmap-based
// sprite rendering: games that declare a charmap have their PUA codepoints
// rendered as sprites from a sprite sheet instead of terminal glyphs.
//
// Use --no-gui for terminal mode: renders as ANSI to the current terminal.
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"strings"

	"github.com/charmbracelet/colorprofile"
	xterm "github.com/charmbracelet/x/term"

	"dev-null/internal/client"
	"dev-null/internal/datadir"
	"dev-null/internal/display"
	"dev-null/internal/engine"
)

//go:embed winres/icon.ico
var appIcon []byte

// buildCommit, buildDate, and buildRemote are injected at build time via -ldflags.
var buildCommit = "dev"
var buildDate = "unknown"
var buildRemote = ""

func main() {
	engine.SetBuildInfo(buildDate, buildRemote)
	host := flag.String("host", "localhost", "server hostname")
	port := flag.Int("port", 23234, "server SSH port")
	player := flag.String("player", defaultPlayer(), "player name")
	noGUI := flag.Bool("no-gui", false, "run in terminal mode (TUI) instead of opening a graphical window")
	gameName := flag.String("game", "", "game to load on connect (sends /game-load command)")
	resumeName := flag.String("resume", "", "game/save to resume on connect, e.g. orbits/autosave (sends /game-resume command)")
	password := flag.String("password", "", "admin password (authenticates as admin on connect)")
	termFlag := flag.String("term", "", "force terminal color profile: truecolor, 256color, ansi, ascii")
	flag.Parse()

	// Build init commands from flags.
	var initCommands []string
	if *resumeName != "" {
		if !strings.Contains(*resumeName, "/") {
			fmt.Fprintf(os.Stderr, "--resume requires game/save format, e.g. orbits/autosave\n")
			os.Exit(1)
		}
		initCommands = append(initCommands, "/game-resume "+*resumeName)
	} else if *gameName != "" {
		initCommands = append(initCommands, "/game-load "+*gameName)
	}

	fmt.Printf("Connecting to %s:%d as %s...\n", *host, *port, *player)
	ptyW, ptyH := 0, 0
	if *noGUI {
		ptyW, ptyH, _ = xterm.GetSize(os.Stdin.Fd())
	}
	conn, err := client.Dial(*host, *port, *player, *noGUI, *termFlag, *password, ptyW, ptyH, initCommands)
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
	if err := display.RunWindow(renderer, "dev-null", 1200, 800, appIcon); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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
