// Package main is a minimal SSH rendering testbed.
//
// It starts a wish+bubbletea SSH server and connects back to it locally,
// reproducing the same pipeline as --local mode without any product code.
// Use it to iterate on ONLCR/staircase rendering fixes in isolation.
//
// Usage:
//
//	go run ./testbed              # SSH mode, no ONLCR fix (expect staircase on Unix)
//	go run ./testbed --onlcr      # SSH mode, with ONLCR fix applied
//	go run ./testbed --no-ssh     # direct Bubble Tea, no SSH (baseline, no artifacts)
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	tea "charm.land/bubbletea/v2"
	"charm.land/wish/v2"
	wishbubbletea "charm.land/wish/v2/bubbletea"
	gossh "github.com/charmbracelet/ssh"
	xterm "github.com/charmbracelet/x/term"
	cryptossh "golang.org/x/crypto/ssh"
)

var (
	flagPort  = flag.Int("port", 22222, "SSH server port")
	flagONLCR = flag.Bool("onlcr", false, "wrap session output with ONLCRWriter (ONLCR fix)")
	flagNoSSH = flag.Bool("no-ssh", false, "skip SSH entirely, run model directly against terminal")
)

func main() {
	flag.Parse()

	if *flagNoSSH {
		runDirect()
		return
	}
	runSSH()
}

// runDirect runs the model directly against os.Stdin/os.Stdout — no server,
// no SSH, no io.Copy. This is the baseline: artifacts here mean the model
// itself is broken; no artifacts here confirm the bug is in the SSH pipeline.
func runDirect() {
	restoreOutput := configureOutput()
	defer restoreOutput()
	defer fmt.Fprint(os.Stdout, "\x1b[?1049l\x1b[?25h")

	program := tea.NewProgram(model{},
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stdout),
		tea.WithFPS(60),
	)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// runSSH starts a minimal wish SSH server and connects back to it locally,
// reproducing the --local pipeline: SSH → io.Copy → os.Stdout.
func runSSH() {
	hostKeyPath := hostKeyFile()
	defer os.Remove(hostKeyPath)

	// Build the wish middleware: model + optional ONLCR wrapper.
	middleware := wishbubbletea.Middleware(func(sess gossh.Session) (tea.Model, []tea.ProgramOption) {
		opts := wishbubbletea.MakeOptions(sess)
		if *flagONLCR {
			// Override the output set by MakeOptions with our ONLCR wrapper.
			opts = append(opts, tea.WithOutput(NewONLCRWriter(sess)))
		}
		return model{}, append(opts, tea.WithFPS(60))
	})

	srv, err := wish.NewServer(
		gossh.EmulatePty(),
		wish.WithAddress(fmt.Sprintf("127.0.0.1:%d", *flagPort)),
		wish.WithHostKeyPath(hostKeyPath),
		wish.WithPasswordAuth(func(_ gossh.Context, _ string) bool { return true }),
		wish.WithMiddleware(middleware),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wish.NewServer: %v\n", err)
		os.Exit(1)
	}

	// Start listening.
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *flagPort))
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go srv.Serve(ln) //nolint:errcheck

	// Get initial terminal size from stdout.
	w, h, err := xterm.GetSize(os.Stdout.Fd())
	if err != nil {
		w, h = 120, 50
	}

	// Dial back to our own server.
	conn, sess, stdout, err := dialSSH(*flagPort, w, h)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SSH dial: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	defer sess.Close()
	defer fmt.Fprint(os.Stdout, "\x1b[?1049l\x1b[?25h")

	// Put local terminal in raw mode (best-effort — skipped if stdin is not a TTY).
	if oldState, err := xterm.MakeRaw(os.Stdin.Fd()); err == nil {
		defer xterm.Restore(os.Stdin.Fd(), oldState)
	}

	// Configure stdout (Windows: enable VT, disable auto-CRLF).
	restoreOutput := configureOutput()
	defer restoreOutput()

	// Forward resize events.
	stdin, _ := sess.StdinPipe()
	stopResize := watchResize(func(w, h int) {
		sess.WindowChange(h, w) //nolint:errcheck
	})
	defer stopResize()

	// Bidirectional pipe: SSH ↔ terminal.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(os.Stdout, stdout) //nolint:errcheck
	}()
	go func() {
		defer wg.Done()
		io.Copy(stdin, os.Stdin) //nolint:errcheck
	}()

	wg.Wait()
	_ = ctx
}

// dialSSH connects to our local SSH server and requests a PTY.
// Returns the client, session, a reader for session stdout, and any error.
func dialSSH(port, w, h int) (*cryptossh.Client, *cryptossh.Session, io.Reader, error) {
	cfg := &cryptossh.ClientConfig{
		User:            "player",
		Auth:            []cryptossh.AuthMethod{cryptossh.Password("")},
		HostKeyCallback: cryptossh.InsecureIgnoreHostKey(), //nolint:gosec
	}

	tcpConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("tcp dial: %w", err)
	}

	c, chans, reqs, err := cryptossh.NewClientConn(tcpConn, "", cfg)
	if err != nil {
		tcpConn.Close()
		return nil, nil, nil, fmt.Errorf("SSH handshake: %w", err)
	}
	client := cryptossh.NewClient(c, chans, reqs)

	sess, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, nil, nil, fmt.Errorf("new session: %w", err)
	}

	// Request PTY with actual terminal dimensions.
	modes := cryptossh.TerminalModes{
		cryptossh.ECHO:  0,
		cryptossh.IGNCR: 1,
		cryptossh.ONLCR: 0,
		cryptossh.OPOST: 0,
	}
	if err := sess.RequestPty("xterm-256color", h, w, modes); err != nil {
		sess.Close()
		client.Close()
		return nil, nil, nil, fmt.Errorf("RequestPty: %w", err)
	}

	// Get stdout pipe before starting the shell.
	stdout, err := sess.StdoutPipe()
	if err != nil {
		sess.Close()
		client.Close()
		return nil, nil, nil, fmt.Errorf("StdoutPipe: %w", err)
	}

	if err := sess.Shell(); err != nil {
		sess.Close()
		client.Close()
		return nil, nil, nil, fmt.Errorf("Shell: %w", err)
	}

	return client, sess, stdout, nil
}

// hostKeyFile creates a temporary file path for the SSH host key.
// wish.WithHostKeyPath auto-generates the key if the file doesn't exist.
func hostKeyFile() string {
	f, err := os.CreateTemp("", "testbed-hostkey-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create temp: %v\n", err)
		os.Exit(1)
	}
	path := f.Name()
	f.Close()
	os.Remove(path) // wish will create it fresh
	return path
}
