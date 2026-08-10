package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func (rm *rcloneManager) setupTray(fyneApp fyne.App) {
	desk, ok := fyneApp.(desktop.App)
	if !ok {
		return // no system tray support on this platform/build
	}
	// SetSystemTrayWindow gives left-click = show/hide the window (like a
	// normal Windows tray icon), right-click = the menu below.
	desk.SetSystemTrayWindow(rm.win)
	desk.SetSystemTrayMenu(rm.buildTrayMenu())
}

// buildTrayMenu mirrors the Python version's _build_tray_menu(): 열기, then
// one toggleable item per configured mount showing ▶(멈춤)/■(실행중) and
// its drive/remote, then 종료.
//
// Every callback here runs fyne.Do(...) — tray/menu clicks arrive on the
// OS's own thread, not Fyne's main loop, so touching widgets or state
// directly from here is exactly the kind of unsafe cross-thread call Fyne
// warns about (and was the real cause of the table/tray going out of sync
// with what was actually mounted).
func (rm *rcloneManager) buildTrayMenu() *fyne.Menu {
	items := []*fyne.MenuItem{
		fyne.NewMenuItem("🪟 열기", func() { fyne.Do(func() { rm.win.Show() }) }),
		fyne.NewMenuItemSeparator(),
	}

	if len(rm.cfg.Mounts) == 0 {
		empty := fyne.NewMenuItem("(등록된 마운트 없음)", nil)
		empty.Disabled = true
		items = append(items, empty, fyne.NewMenuItemSeparator())
	} else {
		for _, m := range rm.cfg.Mounts {
			m := m // capture per-iteration copy for the closure below
			display := trayDisplayLabel(m, rm.isRunning(m.ID))
			items = append(items, fyne.NewMenuItem(display, func() {
				fyne.Do(func() { rm.toggleFromTray(m) })
			}))
		}
		items = append(items, fyne.NewMenuItemSeparator())
	}

	items = append(items, fyne.NewMenuItem("🚪 종료", func() {
		fyne.Do(func() { fyne.CurrentApp().Quit() })
	}))

	return fyne.NewMenu("RcloneManager", items...)
}

// toggleFromTray mounts or unmounts m, and fires a desktop notification —
// the menu closes the instant you click an item, so without this there's
// no way to tell whether the click actually did anything.
func (rm *rcloneManager) toggleFromTray(m engine.Mount) {
	if rm.isRunning(m.ID) {
		rm.notify(trayShortLabel(m) + " 해제 요청됨")
		rm.unmount(m.ID)
	} else {
		rm.notify(trayShortLabel(m) + " 마운트 시작됨")
		rm.mount(m)
	}
}

func (rm *rcloneManager) notify(message string) {
	fyne.CurrentApp().SendNotification(fyne.NewNotification("RcloneManager", message))
}

// refreshTrayMenu rebuilds and re-applies the tray menu — call this
// whenever mount state or the mount list itself changes, since Fyne's
// menu is a static snapshot rather than something that reflects live
// state on its own (mirrors the Python version's `_tray.menu = ...;
// update_menu()` after every _refresh_list()).
func (rm *rcloneManager) refreshTrayMenu() {
	desk, ok := fyne.CurrentApp().(desktop.App)
	if !ok {
		return
	}
	desk.SetSystemTrayMenu(rm.buildTrayMenu())
}

// trayShortLabel is the mount's short display name: its drive letter if
// set, else the remote name.
func trayShortLabel(m engine.Mount) string {
	if label := strings.TrimSpace(m.Drive); label != "" {
		return label
	}
	return m.Remote
}

// trayDisplayLabel is the full tray menu item text for one mount. Pulled
// out as a pure function for testing — see tray_test.go.
func trayDisplayLabel(m engine.Mount, running bool) string {
	icon := "▶"
	if running {
		icon = "■"
	}
	rstr := strings.TrimRight(fmt.Sprintf("%s:%s", m.Remote, m.RemotePath), ":")
	return fmt.Sprintf("%s  %s  (%s)", icon, trayShortLabel(m), rstr)
}
