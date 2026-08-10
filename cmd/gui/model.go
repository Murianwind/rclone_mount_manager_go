package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

// runningMount tracks a live rclone mount process. done is closed by the
// single goroutine that owns cmd.Wait() — unmount() waits on it (with a
// timeout) instead of calling Wait() itself, since exec.Cmd.Wait() may
// only be called once. stderr captures rclone's error output so a failed
// mount can show *why* it failed instead of just going quietly back to
// "해제됨". stoppedByUs distinguishes a failure from a normal unmount
// (both make the process exit, often with a non-zero code).
type runningMount struct {
	cmd         *exec.Cmd
	done        chan struct{}
	stderr      *bytes.Buffer
	stoppedByUs bool
}

// rcloneManager is the single owner of all app state — config, running
// mount processes, and the widgets that need to be refreshed when either
// changes. One instance is created in main() and threaded through every
// UI callback and background goroutine.
type rcloneManager struct {
	appDir string
	log    engine.RotatingLog
	store  engine.Store
	cfg    engine.Config
	win    fyne.Window

	table         *widget.Table
	rcPathEntry   *widget.Entry
	rcVersionText *widget.Button // clickable — tapping checks for a newer rclone

	selectedRow int // -1 = nothing selected; used by the 위/아래 이동 buttons

	latestRcloneVersion string // cached from the last successful check; "" = unknown

	activeMu sync.Mutex
	active   map[string]*runningMount
}

func newRcloneManager(appDir string, log engine.RotatingLog, win fyne.Window) *rcloneManager {
	return &rcloneManager{
		appDir:      appDir,
		log:         log,
		store:       engine.Store{Dir: appDir, Log: func(level, msg string) { _ = log.Write(level, msg) }},
		win:         win,
		active:      map[string]*runningMount{},
		selectedRow: -1,
	}
}

// logf writes one line to RcloneManager.log. Logging failures are
// deliberately ignored here (never let a broken log stop the app) — same
// intent as the Python version's write_log().
func (rm *rcloneManager) logf(level, format string, args ...any) {
	_ = rm.log.Write(level, fmt.Sprintf(format, args...))
}

func (rm *rcloneManager) isRunning(mountID string) bool {
	rm.activeMu.Lock()
	defer rm.activeMu.Unlock()
	_, running := rm.active[mountID]
	return running
}

// persist saves the current config to mounts.json and refreshes every
// view that shows mount state (the table and the tray menu) — the single
// place both need to stay in sync, so nothing calls store.Save directly.
func (rm *rcloneManager) persist() {
	if err := rm.store.Save(rm.cfg); err != nil {
		rm.logf("ERROR", "[설정] mounts.json 저장 실패: %v", err)
		rm.revealWindow()
		dialog.ShowError(err, rm.win)
		return
	}
	rm.table.Refresh()
	rm.refreshTrayMenu()
}

// saveWindowSize records the window's current size so it's restored on
// the next launch. Called on hide-to-tray and before quitting — the app
// never has a plain "on close" event otherwise, since the titlebar X is
// intercepted to hide rather than close.
func (rm *rcloneManager) saveWindowSize() {
	size := rm.win.Canvas().Size()
	rm.cfg.WindowWidth = size.Width
	rm.cfg.WindowHeight = size.Height
	rm.persist()
}

// revealWindow brings the main window to the front regardless of the
// "시작 시 트레이로 최소화" setting or its current hidden/minimized
// state. Call this right before showing any dialog that represents a
// real problem (a failed mount, a save error, ...) — a dialog rendered
// on a hidden window's canvas is itself invisible, so without this a
// user running minimized-to-tray would never know anything went wrong.
func (rm *rcloneManager) revealWindow() {
	rm.win.Show()
	rm.win.RequestFocus()
}
