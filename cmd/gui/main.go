// Command gui is RcloneManager's Windows desktop app: a Fyne front end
// over the internal/engine package.
//
// File layout (by concern, not by widget type):
//
//	main.go     - entrypoint, app/window bootstrap
//	model.go    - rcloneManager state, small shared helpers (logf, persist, isRunning)
//	layout.go   - top-level window layout (header, path row, options row)
//	table.go    - the mount list table
//	dialogs.go  - add/edit/delete/mount-failure dialogs
//	mount.go    - mount/unmount process lifecycle, auto-mount, network monitor
//	update.go   - self-update check/download/apply
//	tray.go     - system tray icon + menu
package main

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

const appVersion = "0.0.4"
const issueURL = "https://github.com/Murianwind/rclone_mount_manager_go/issues/new"

func main() {
	appDir := mustAppDir()
	log := engine.RotatingLog{Path: filepath.Join(appDir, "RcloneManager.log"), MaxLines: 1000}

	fyneApp := app.NewWithID("com.murianwind.rclonemanager")
	win := fyneApp.NewWindow("RcloneManager")
	win.Resize(fyne.NewSize(760, 480))

	rm := newRcloneManager(appDir, log, win)
	rm.logf("INFO", "[시작] RcloneManager v%s 시작됨", appVersion)

	cfg, err := rm.store.Load()
	if err != nil {
		rm.logf("ERROR", "[설정] mounts.json 로드 실패: %v", err)
		dialog.ShowError(err, win)
	}
	rm.cfg = cfg

	rm.build()
	rm.setupTray(fyneApp)
	rm.refreshVersionLabel()
	rm.startNetworkMonitor()

	win.SetCloseIntercept(func() {
		fyne.Do(func() { win.Hide() }) // minimize to tray instead of quitting
	})

	// Auto-mount and the update check both need the event loop actually
	// running (dialogs/UI updates aren't safe before that), so they're
	// wired to the "app started" hook rather than called directly here.
	fyneApp.Lifecycle().SetOnStarted(func() {
		rm.autoMountAll()
		rm.checkForUpdate(false)
	})

	if rm.cfg.StartMinimized {
		// Deliberately skip win.Show(): ShowAndRun() would force it open
		// regardless. The tray icon (already wired in setupTray) keeps the
		// app reachable.
		fyneApp.Run()
	} else {
		win.Show()
		fyneApp.Run()
	}
}

// mustAppDir returns the directory the running executable lives in — the
// Go equivalent of the Python version's APP_DIR. mounts.json, rclone.exe,
// and the log file all live here.
func mustAppDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}
