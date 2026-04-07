// Package clipboard provides a simple cross-platform clipboard copy.
package clipboard

import (
	"os/exec"
	"runtime"
	"strings"
)

// Copy writes text to the system clipboard.
// On Windows: clip.exe, macOS: pbcopy, Linux: xclip.
func Copy(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("clip.exe")
	case "darwin":
		cmd = exec.Command("pbcopy")
	default:
		cmd = exec.Command("xclip", "-selection", "clipboard")
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
