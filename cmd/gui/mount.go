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

// mount starts rclone for m, unless it's already running. Safe to call
// from any goroutine — all UI touches are wrapped in fyne.Do.
func (rm *rcloneManager) mount(m engine.Mount) {
	if rm.isRunning(m.ID) {
		return // already mounted — avoid spawning a duplicate rclone process
	}

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
	engine.ConfigureBackgroundProcess(cmd) // hide console window, own process group
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	if err := cmd.Start(); err != nil {
		rm.logf("ERROR", "[마운트] %s:%s 프로세스 시작 실패: %v", m.Remote, m.RemotePath, err)
		fyne.Do(func() {
			rm.revealWindow()
			dialog.ShowError(err, rm.win)
		})
		return
	}
	rm.logf("INFO", "[마운트] %s:%s → %s 시작 (pid %d)", m.Remote, m.RemotePath, m.Drive, cmd.Process.Pid)

	done := make(chan struct{})
	rm.activeMu.Lock()
	rm.active[m.ID] = &runningMount{cmd: cmd, done: done, stderr: &stderrBuf}
	rm.activeMu.Unlock()
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

// stopMountAndWait gracefully stops a running mount (so rclone unmounts
// its WinFsp filesystem cleanly instead of leaving the drive letter
// stuck), falling back to a hard Kill() if it doesn't exit in time, and
// blocks until it's actually gone.
//
// This blocking form exists specifically for app shutdown: quitting
// without waiting here left rclone.exe running as an orphaned process
// still holding the drive, which is why every next launch failed with
// "mountpoint path already exists" — nothing had ever waited for the
// unmount to actually finish before the app exited.
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
		return
	}
	select {
	case <-running.done:
		// exited cleanly on its own
	case <-time.After(5 * time.Second):
		rm.logf("WARN", "[언마운트] 5초 내 종료 안 됨 → 강제 종료")
		_ = running.cmd.Process.Kill()
	}
}

// quitGracefully unmounts everything currently active and waits for it to
// finish before actually quitting the app. This is the only place that
// should ever call fyne.CurrentApp().Quit() — quitting directly (as the
// tray "종료" item used to) left rclone.exe processes orphaned and still
// holding their drives, which broke every subsequent launch.
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

// autoMountAll starts every mount flagged AutoMount. Called once from the
// app's "started" lifecycle hook so it runs after the event loop (and
// therefore dialogs/UI updates) is actually safe to use — and again by the
// network monitor whenever connectivity is (re)established.
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
// (quitGracefully) or restarts itself (self-update) — both need the
// drives released before the process disappears, not just "asked to
// release."
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
// state *transition* (disconnected->connected or vice versa) — mirrors the
// Python version's _start_net_monitor exactly, including starting from an
// "unknown" state so the very first check always fires once.
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
