package daemon

import (
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const daemonEnv = "XCONNECT_DAEMON=1"

// IsDaemonChild returns true if this process was started as the daemon child (do not fork again).
func IsDaemonChild() bool {
	return os.Getenv("XCONNECT_DAEMON") == "1"
}

// RunInBackground starts this program again in the background with stdout/stderr redirected to logPath,
// and exits the current process. The child will have XCONNECT_DAEMON=1 set so it does not fork again.
// logPath is used for the child's stdout/stderr; if empty, DefaultLogPath() is used.
func RunInBackground(logPath string) error {
	if logPath == "" {
		logPath = DefaultLogPath()
	}
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer logFile.Close()

	self, err := os.Executable()
	if err != nil {
		return err
	}
	args := os.Args[1:]
	env := append(os.Environ(), daemonEnv)

	cmd := exec.Command(self, args...)
	cmd.Env = env
	cmd.Stdin = nil
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	switch runtime.GOOS {
	case "windows":
		cmd.SysProcAttr = sysProcAttrWindows()
	default:
		cmd.SysProcAttr = sysProcAttrUnix()
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	_ = cmd.Process.Release()
	log.Printf("xconnect daemon started (PID %d), logs: %s", cmd.Process.Pid, logPath)
	os.Exit(0)
	return nil
}

// RedirectLogToFile sets log output to the given file (and optionally stderr).
// Call this in the daemon child so all log output goes to the file.
func RedirectLogToFile(w io.Writer, alsoStderr bool) {
	if alsoStderr {
		log.SetOutput(io.MultiWriter(os.Stderr, w))
	} else {
		log.SetOutput(w)
	}
	log.SetFlags(log.Ldate | log.Ltime)
}
