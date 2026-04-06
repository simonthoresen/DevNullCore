//go:build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"

	xterm "github.com/charmbracelet/x/term"
)

// configureOutput is a no-op on Unix — MakeRaw already disables ONLCR output
// processing via the terminal's c_oflag.
func configureOutput() func() { return func() {} }

// watchResize listens for SIGWINCH and forwards window size changes.
func watchResize(onChange func(w, h int)) func() {
	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)
	go func() {
		for range sigwinch {
			if w, h, err := xterm.GetSize(os.Stdout.Fd()); err == nil {
				onChange(w, h)
			}
		}
	}()
	return func() {
		signal.Stop(sigwinch)
		close(sigwinch)
	}
}
