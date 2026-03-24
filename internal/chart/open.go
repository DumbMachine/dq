package chart

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenBrowser opens the given file path in the default browser.
func OpenBrowser(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", path)
	default:
		return fmt.Errorf("unsupported platform %s — open %s manually", runtime.GOOS, path)
	}
	return cmd.Start()
}
