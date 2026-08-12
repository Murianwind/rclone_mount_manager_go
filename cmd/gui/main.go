// Command gui is RcloneManager's Windows desktop app: a Fyne front end
// over the internal/engine package.
//
// File layout (by concern, not by widget type):
//
//	main.go         - entrypoint, app/window bootstrap
//	model.go        - rcloneManager state, small shared helpers (logf, persist, isRunning)
//	rows.go         - combined 원본/마운트 table row model
//	layout.go       - top-level window layout (header, path row, options row)
//	table.go        - the mount list table
//	dialogs.go      - add/edit/delete/mount-failure dialogs
//	confimport.go   - rclone.conf import dialog
//	reorder.go      - 위/아래 list reordering
//	mount.go        - mount/unmount process lifecycle, auto-mount, network monitor
//	update.go       - app self-update check/download/apply
//	rcloneupdate.go - rclone.exe install/update check/download/apply
//	tray.go         - system tray icon + menu
package main

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"

	"github.com/murianwind/rclone-manager-go/internal/engine"
)

const appVersion = "1.0.1"
const issueURL = "https://github.com/Murianwind/rclone_mount_manager_go/issues/new"

// 컬럼 합(약 686px) + 창 여백/스크롤바를 감안해 기본 창 너비에 여유를 둔다 —
// 정확히 맞추면 창 테두리에 마지막 컬럼이 잘려 보인다.
const defaultWindowWidth = 820
const defaultWindowHeight = 520

func main() {
	appDir := mustAppDir()
	log := engine.RotatingLog{Path: filepath.Join(appDir, "RcloneManager.log"), MaxLines: 1000}

	if exe, err := os.Executable(); err == nil {
		engine.CleanupPreviousExe(exe)
	}

	fyneApp := app.NewWithID("com.murianwind.rclonemanager")
	fyneApp.SetIcon(appIcon)
	fyneApp.Settings().SetTheme(newWindowsFontTheme(fyneApp.Settings().Theme()))
	win := fyneApp.NewWindow("RcloneManager")
	win.SetIcon(appIcon)

	rm := newRcloneManager(appDir, log, win)
	rm.logf("INFO", "[시작] RcloneManager v%s 시작됨", appVersion)

	// 등록된 시작프로그램 경로가 지금 실행 중인 exe와 다르면(구버전에서
	// 옮겨왔거나 폴더를 이동한 경우 등) 조용히 재등록한다 — 체크박스는
	// "등록됨"으로 보이는데 실제로는 존재하지 않는/옛날 경로를 가리키는
	// 상태로 방치되는 걸 막기 위함.
	if fixed, err := engine.CheckAndFixStartup(); err != nil {
		rm.logf("WARN", "[시작프로그램] 경로 재등록 확인 실패: %v", err)
	} else if fixed {
		rm.logf("INFO", "[시작프로그램] 등록된 경로가 실행 파일과 달라 재등록함")
	}

	cfg, err := rm.store.Load()
	if err != nil {
		rm.logf("ERROR", "[설정] mounts.json 로드 실패: %v", err)
		rm.revealWindow()
		dialog.ShowError(err, win)
	}
	rm.cfg = cfg

	win.Resize(fyne.NewSize(savedOr(cfg.WindowWidth, defaultWindowWidth), savedOr(cfg.WindowHeight, defaultWindowHeight)))

	rm.build()
	rm.setupTray(fyneApp)
	rm.refreshVersionLabel()
	enforceMinWindowSize(win)

	win.SetCloseIntercept(func() {
		fyne.Do(func() {
			rm.saveWindowSize()
			rm.selectedRow = -1
			rm.table.Refresh()
			win.Hide() // minimize to tray instead of quitting
		})
	})

	// Auto-mount is initiated by startNetworkMonitor's first connectivity
	// check. Starting the monitor here would race with the app-started hook,
	// so start it only after the Fyne event loop is live.
	// The monitor treats its first successful connectivity check as the
	// initial transition and therefore performs exactly one auto-mount pass.
	fyneApp.Lifecycle().SetOnStarted(func() {
		rm.startNetworkMonitor()
		rm.checkForUpdate(false)
		rm.checkRcloneUpdate(false)
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

// savedOr returns saved if it's a usable positive size, else fallback.
// Pulled out as a pure function so the "0 means unset" rule is testable
// without a running Fyne window.
func savedOr(saved, fallback float32) float32 {
	if saved > 0 {
		return saved
	}
	return fallback
}
