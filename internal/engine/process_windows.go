//go:build windows

package engine

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// ConfigureBackgroundProcess hides the console window while giving the child
// its own console and process group. GenerateConsoleCtrlEvent can only deliver
// CTRL_BREAK to a process group that shares the caller's console; a -H=windowsgui
// parent has no console of its own, so the child needs a private hidden one.
// Must be called before cmd.Start().
func ConfigureBackgroundProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NEW_CONSOLE,
	}
}

// SignalGracefulStop asks a rclone process to shut down cleanly by sending
// CTRL_BREAK_EVENT. A GUI parent normally has no console, so the first direct
// attempt is expected to fail. In that case temporarily attach to the child's
// hidden console, send the signal, then detach again.
func SignalGracefulStop(pid int) error {
	err := windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(pid))
	if err == nil {
		return nil
	}
	firstErr := err

	if attachErr := windows.AttachConsole(uint32(pid)); attachErr != nil {
		return firstErr
	}
	defer windows.FreeConsole()

	return windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(pid))
}
