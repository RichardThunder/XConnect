package daemon

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// DefaultLogPath returns a platform-specific default path for the log file.
func DefaultLogPath() string {
	switch runtime.GOOS {
	case "windows":
		dir := os.Getenv("LocalAppData")
		if dir == "" {
			dir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
		return filepath.Join(dir, "XConnect", "logs", "xconnect.log")
	default:
		dir := os.Getenv("XDG_STATE_HOME")
		if dir == "" {
			dir = filepath.Join(os.Getenv("HOME"), ".local", "state")
		}
		return filepath.Join(dir, "xconnect", "xconnect.log")
	}
}

// SetupLog opens the log file (creating parent dirs), sets log output to it and optionally stderr.
// If logPath is empty, uses DefaultLogPath(). Returns the opened file (caller may defer f.Close()).
func SetupLog(logPath string, alsoStderr bool) (*os.File, error) {
	if logPath == "" {
		logPath = DefaultLogPath()
	}
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	var w io.Writer = f
	if alsoStderr {
		w = io.MultiWriter(os.Stderr, f)
	}
	log.SetOutput(w)
	log.SetFlags(log.Ldate | log.Ltime)
	return f, nil
}
