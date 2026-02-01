//go:build windows

package daemon

import (
	"syscall"
)

func sysProcAttrWindows() *syscall.SysProcAttr {
	const (
		DETACHED_PROCESS = 0x00000008
		CREATE_NO_WINDOW = 0x08000000
	)
	return &syscall.SysProcAttr{
		CreationFlags: DETACHED_PROCESS | CREATE_NO_WINDOW,
	}
}

func sysProcAttrUnix() *syscall.SysProcAttr {
	return nil
}
