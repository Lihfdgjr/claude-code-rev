package oauth

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenBrowser opens the default browser at url. Best-effort; returns an
// error if the underlying launcher cannot be started.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/C", "start", "", url)
	default:
		return fmt.Errorf("oauth: unsupported platform %q", runtime.GOOS)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("oauth: open browser: %w", err)
	}
	return nil
}
