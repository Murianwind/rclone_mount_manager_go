//go:build windows

package engine

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// ConfigureBackgroundProcess hides the console window a Windows child
// process would otherwise pop up, and puts it in its own process group so
// SignalGracefulStop can later ask it to shut down cleanly. Mirrors the
// Python version's _CREATE_NO_WINDOW / _DETACHED_PROCESS flags. Must be
// called before cmd.Start().
func ConfigureBackgroundProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP,
	}
}

// SignalGracefulStop asks a process started via ConfigureBackgroundProcess
// to shut down cleanly by sending CTRL_BREAK_EVENT. rclone handles this by
// unmounting its WinFsp filesystem before exiting — unlike a hard Kill(),
// which can leave the mountpoint stuck (the "mountpoint path already
// exists" error). Callers should still fall back to Process.Kill() after a
// timeout in case the process doesn't respond.
func SignalGracefulStop(pid int) error {
	return windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(pid))
}
