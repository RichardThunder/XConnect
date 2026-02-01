package clipboard

import (
	"fmt"
	"runtime"

	"github.com/atotto/clipboard"
)

// ReadAll returns the clipboard content, or a user-friendly error including install hints.
func ReadAll() (string, error) {
	s, err := clipboard.ReadAll()
	if err != nil {
		return "", fmt.Errorf("%w\n%s", err, installHint("read"))
	}
	return s, nil
}

// WriteAll writes content to the clipboard, or returns a user-friendly error including install hints.
func WriteAll(content string) error {
	if err := clipboard.WriteAll(content); err != nil {
		return fmt.Errorf("%w\n%s", err, installHint("write"))
	}
	return nil
}

// installHint returns platform-specific install instructions for clipboard tools.
func installHint(op string) string {
	switch runtime.GOOS {
	case "linux":
		return `Linux clipboard: install one of:
  • xclip:   sudo dnf install xclip       (Fedora/RHEL)
             sudo apt install xclip       (Debian/Ubuntu)
             sudo pacman -S xclip         (Arch)
  • xsel:    sudo dnf install xsel
  • Wayland: sudo dnf install wl-clipboard   (Fedora)
             sudo apt install wl-clipboard   (Debian/Ubuntu)
  Or run: ./scripts/install-clipboard-deps.sh`
	case "darwin":
		return "macOS: clipboard access usually works. If not, ensure the app has accessibility permissions."
	case "windows":
		return "Windows: clipboard uses system API; no extra install. If it fails, run as the logged-in user (not a service)."
	default:
		return "Install a clipboard utility for your OS (xclip, xsel, or wl-clipboard on Linux)."
	}
}
