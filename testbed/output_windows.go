//go:build windows

package main

import (
	"os"
	"time"

	xterm "github.com/charmbracelet/x/term"
	"golang.org/x/sys/windows"
)

// configureOutput enables VT processing and disables auto-CRLF on stdout.
// Without DISABLE_NEWLINE_AUTO_RETURN, Windows maps \n → \r\n automatically,
// which resets the cursor to col 0. Bubble Tea v2's renderer (mapNl=false on
// Windows) tracks fx through \n without resetting, so the unexpected \r causes
// characters to land at col 0 instead of the correct column.
func configureOutput() func() {
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

// watchResize polls for terminal size changes (Windows has no SIGWINCH).
func watchResize(onChange func(w, h int)) func() {
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
					onChange(w, h)
				}
			}
		}
	}()
	return func() { close(done) }
}
