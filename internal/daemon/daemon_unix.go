//go:build !windows

package daemon

import "syscall"

func sysProcAttrWindows() *syscall.SysProcAttr {
	return nil
}

func sysProcAttrUnix() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}
