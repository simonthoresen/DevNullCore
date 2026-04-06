//go:build windows

package server

import (
	"os"
	"time"

	xterm "github.com/charmbracelet/x/term"
	"golang.org/x/sys/windows"

	"dev-null/internal/client"
)

// configureLocalOutput enables VT processing and disables auto-CRLF on
// os.Stdout so that ANSI sequences written by io.Copy in RunLocalSSH render
// correctly. Without this, Windows applies ONLCR (\n → \r\n) to the SSH
// output stream, which corrupts Bubble Tea's cursor tracking.
// Returns a restore function that reverts to the original console mode.
func configureLocalOutput() func() {
	handle := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return func() {}
	}
	newMode := mode | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING | windows.DISABLE_NEWLINE_AUTO_RETURN
	if err := windows.SetConsoleMode(handle, newMode); err != nil {
		return func() {}
	}
	return func() { windows.SetConsoleMode(handle, mode) } //nolint:errcheck
}

// watchTerminalResize polls for terminal size changes on Windows (no SIGWINCH).
// Returns a stop function to clean up.
func watchTerminalResize(conn *client.SSHConn) func() {
	done := make(chan struct{})
	go func() {
		lastW, lastH, _ := xterm.GetSize(os.Stdout.Fd())
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if w, h, err := xterm.GetSize(os.Stdout.Fd()); err == nil && (w != lastW || h != lastH) {
					lastW, lastH = w, h
					conn.SendWindowChange(w, h)
				}
			}
		}
	}()
	return func() { close(done) }
}
