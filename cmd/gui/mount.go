package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

// mount starts rclone for m, unless the mount entry is already reserved by a
// live or starting process. Reservation happens before cmd.Start(), so two
// concurrent callers cannot both pass the isRunning check and spawn duplicate
// rclone processes for the same mount.
func (rm *rcloneManager) mount(m engine.Mount) {
	exe, ok := rm.rcloneExePath()
	if !ok {
		rm.logf("ERROR", "[마운트] %s:%s 실패 — rclone.exe를 찾을 수 없음", m.Remote, m.RemotePath)
		fyne.Do(func() {
			rm.revealWindow()
			dialog.ShowInformation("알림", "rclone.exe를 찾을 수 없습니다. 먼저 rclone 경로를 등록해 주세요.", rm.win)
		})
		return
	}

	args := engine.BuildCmd(exe, m)
	cmd := exec.Command(args[0], args[1:]...)
	engine.ConfigureBackgroundProcess(cmd) // hide console window, own process group/console
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	done := make(chan struct{})
	running := &runningMount{cmd: cmd, done: done, stderr: &stderrBuf}

	// Reserve the mount before Start(). This closes the race between
	// startup auto-mount, network-monitor transitions, and manual mounting.
	rm.activeMu.Lock()
	if _, exists := rm.active[m.ID]; exists {
		rm.activeMu.Unlock()
		return
	}
	rm.active[m.ID] = running
	rm.activeMu.Unlock()

	if err := cmd.Start(); err != nil {
		rm.activeMu.Lock()
		if rm.active[m.ID] == running {
			delete(rm.active, m.ID)
		}
		rm.activeMu.Unlock()
		rm.logf("ERROR", "[마운트] %s:%s 프로세스 시작 실패: %v", m.Remote, m.RemotePath, err)
		fyne.Do(func() {
			rm.revealWindow()
			dialog.ShowError(err, rm.win)
		})
		return
	}

	rm.logf("INFO", "[마운트] %s:%s → %s 시작 (pid %d)", m.Remote, m.RemotePath, m.Drive, cmd.Process.Pid)
	fyne.Do(func() { rm.table.Refresh(); rm.refreshTrayMenu() })

	go rm.waitForMountExit(m, cmd, done, &stderrBuf)
}

// waitForMountExit owns the one legal cmd.Wait() call for this process. It
// tells a genuine mount failure apart from a normal, user-requested
// unmount (both make the process exit, often with a non-zero code) via
// runningMount.stoppedByUs, and only surfaces a failure dialog for the
// former.
func (rm *rcloneManager) waitForMountExit(m engine.Mount, cmd *exec.Cmd, done chan struct{}, stderrBuf *bytes.Buffer) {
	err := cmd.Wait()
	close(done)

	rm.activeMu.Lock()
	running := rm.active[m.ID]
	stoppedByUs := running != nil && running.stoppedByUs
	delete(rm.active, m.ID)
	rm.activeMu.Unlock()

	detail := strings.TrimSpace(stderrBuf.String())
	if err != nil {
		rm.logf("WARN", "[마운트] %s:%s 프로세스 종료 (오류 종료: %v)", m.Remote, m.RemotePath, err)
		if detail != "" {
			rm.logf("ERROR", "[마운트] %s:%s 오류 상세: %s", m.Remote, m.RemotePath, detail)
		}
	} else {
		rm.logf("INFO", "[마운트] %s:%s 프로세스 종료", m.Remote, m.RemotePath)
	}

	fyne.Do(func() {
		rm.table.Refresh()
		rm.refreshTrayMenu()
		if shouldReportMountFailure(err, stoppedByUs) {
			rm.showMountFailureDialog(m, detail)
		}
	})
}

// shouldReportMountFailure decides whether an rclone process exit is worth
// interrupting the user for. Pulled out as a pure function for testing —
// see mount_test.go.
func shouldReportMountFailure(exitErr error, stoppedByUs bool) bool {
	return exitErr != nil && !stoppedByUs
}

// unmount asks a running mount to stop, without blocking the caller (safe
// to call from a UI button handler). See stopMountAndWait for the actual
// stop logic — this just fires it off in a goroutine.
func (rm *rcloneManager) unmount(mountID string) {
	go rm.stopMountAndWait(mountID)
}

// stopMountAndWait gracefully stops a running mount and does not return until
// the rclone process has actually exited. If the graceful signal is rejected
// or the process does not exit in time, it falls back to Kill() and still
// waits for cmd.Wait() to finish. This prevents the application from quitting
// while rclone.exe is still holding a WinFsp mountpoint.
func (rm *rcloneManager) stopMountAndWait(mountID string) {
	rm.activeMu.Lock()
	running, ok := rm.active[mountID]
	if ok {
		running.stoppedByUs = true
	}
	rm.activeMu.Unlock()
	if !ok || running.cmd.Process == nil {
		return
	}

	if err := engine.SignalGracefulStop(running.cmd.Process.Pid); err != nil {
		rm.logf("WARN", "[언마운트] 정상 종료 신호 실패(%v) → 강제 종료", err)
		_ = running.cmd.Process.Kill()
		rm.waitForStoppedProcess(running.done)
		return
	}

	select {
	case <-running.done:
		// rclone handled CTRL_BREAK and completed its WinFsp unmount.
	case <-time.After(5 * time.Second):
		rm.logf("WARN", "[언마운트] 5초 내 종료 안 됨 → 강제 종료")
		_ = running.cmd.Process.Kill()
		rm.waitForStoppedProcess(running.done)
	}
}

// waitForStoppedProcess gives cmd.Wait() time to observe the killed process.
// The goroutine that owns cmd.Wait() closes done, so this never calls Wait a
// second time and therefore cannot race with waitForMountExit.
func (rm *rcloneManager) waitForStoppedProcess(done <-chan struct{}) {
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		rm.logf("ERROR", "[언마운트] 강제 종료 후에도 rclone 프로세스 종료 확인 실패")
	}
}

// quitGracefully unmounts everything currently active and waits for it to
// finish before actually quitting the app. This is the only place that
// should ever call fyne.CurrentApp().Quit() — quitting directly (as the
// tray "종료" item used to) left rclone.exe processes orphaned and still
// holding their drives.
func (rm *rcloneManager) quitGracefully() {
	rm.saveWindowSize()
	active := rm.activeMountsSnapshot()
	if len(active) == 0 {
		fyne.CurrentApp().Quit()
		return
	}
	rm.logf("INFO", "[종료] 마운트 %d개 해제 후 종료", len(active))
	go func() {
		rm.unmountAllAndWait()
		fyne.Do(func() { fyne.CurrentApp().Quit() })
	}()
}

// testMountConnection runs `rclone lsf <remote>:<path> --max-depth 1` to
// verify the remote/path is actually reachable, before the user commits
// to saving the mount. Mirrors the Python version's MountDialog._test().
func (rm *rcloneManager) testMountConnection(remote, path string) {
	exe, ok := rm.rcloneExePath()
	if !ok {
		dialog.ShowInformation("알림", "rclone 경로가 등록되어 있지 않습니다.", rm.win)
		return
	}
	target := remote + ":" + strings.Trim(path, "/")

	go func() {
		cmd := exec.Command(exe, "lsf", target, "--max-depth", "1")
		engine.ConfigureBackgroundProcess(cmd)
		out, err := cmd.CombinedOutput()
		fyne.Do(func() {
			if err == nil {
				dialog.ShowInformation("성공", "연결 확인 완료!", rm.win)
				return
			}
			rm.logf("ERROR", "[연결 테스트] %s 실패: %v", target, err)
			dialog.ShowInformation("연결 실패", fmt.Sprintf("연결 불가:\n%s", strings.TrimSpace(string(out))), rm.win)
		})
	}()
}

// autoMountAll starts every mount flagged AutoMount. The network monitor owns
// the initial connectivity transition as well as later reconnects, so there
// is exactly one startup trigger for this operation.
func (rm *rcloneManager) autoMountAll() {
	for _, m := range rm.cfg.Mounts {
		if m.AutoMount {
			rm.mount(m)
		}
	}
}

// unmountAllOnDisconnect unmounts every currently-active mount — regardless
// of its AutoMount flag — since a mount whose remote just went unreachable
// can hang the drive if left alone.
func (rm *rcloneManager) unmountAllOnDisconnect() {
	rm.activeMu.Lock()
	ids := make([]string, 0, len(rm.active))
	for id := range rm.active {
		ids = append(ids, id)
	}
	rm.activeMu.Unlock()

	for _, id := range ids {
		rm.unmount(id)
	}
}

// unmountAllAndWait stops every currently-active mount concurrently and
// blocks until all of them are actually gone. Used before the app exits
// (quitGracefully) or restarts itself (self-update).
func (rm *rcloneManager) unmountAllAndWait() {
	rm.activeMu.Lock()
	ids := make([]string, 0, len(rm.active))
	for id := range rm.active {
		ids = append(ids, id)
	}
	rm.activeMu.Unlock()

	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(mountID string) {
			defer wg.Done()
			rm.stopMountAndWait(mountID)
		}(id)
	}
	wg.Wait()
}

// startNetworkMonitor polls connectivity every 10s and reacts only on a
// state transition. The first completed check establishes the initial state;
// if connected, that initial transition is the single startup auto-mount.
// It is called from the Fyne app-started lifecycle hook so all UI work is safe.
func (rm *rcloneManager) startNetworkMonitor() {
	go func() {
		var wasConnected *bool
		for {
			connected := engine.IsInternetAvailable("8.8.8.8", 53, 3*time.Second)

			if wasConnected == nil || *wasConnected != connected {
				c := connected
				wasConnected = &c
				if connected {
					rm.autoMountAll()
				} else {
					rm.unmountAllOnDisconnect()
				}
			}

			time.Sleep(10 * time.Second)
		}
	}()
}

// detectLocalRcloneVersion runs `rclone version` and formats the result
// for the version label.
func detectLocalRcloneVersion(exe string) string {
	cmd := exec.Command(exe, "version")
	engine.ConfigureBackgroundProcess(cmd)
	out, err := cmd.CombinedOutput()
	return formatRcloneVersionLabel(string(out), err)
}

func formatRcloneVersionLabel(output string, runErr error) string {
	if runErr == nil {
		if ver, found := engine.ParseLocalRcloneVersion(output); found {
			return "rclone v" + ver
		}
	}
	return "v알 수 없음"
}
