package main

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

const appVersion = "2.0.0"
const issueURL = "https://github.com/Murianwind/rclone_mount_manager_go/issues/new"

// table column indices — keep in sync with buildTable's header labels.
const (
	colAuto = iota
	colDrive
	colRemote
	colStatus
	colActions
	colCount
)

// runningMount tracks a live rclone mount process. done is closed by the
// single goroutine that owns cmd.Wait() — unmount() waits on it (with a
// timeout) instead of calling Wait() itself, since exec.Cmd.Wait() may
// only be called once.
type runningMount struct {
	cmd  *exec.Cmd
	done chan struct{}
}

func main() {
	appDir := mustAppDir()

	fyneApp := app.NewWithID("com.murianwind.rclonemanager")
	win := fyneApp.NewWindow("RcloneManager")
	win.Resize(fyne.NewSize(760, 480))

	rm := &rcloneManager{
		appDir: appDir,
		store:  engine.Store{Dir: appDir, Log: nil},
		win:    win,
		active: map[string]*runningMount{},
	}

	cfg, err := rm.store.Load()
	if err != nil {
		dialog.ShowError(err, win)
	}
	rm.cfg = cfg

	rm.build()
	rm.setupTray(fyneApp)
	rm.refreshVersionLabel()
	rm.startNetworkMonitor()

	win.SetCloseIntercept(func() {
		win.Hide() // minimize to tray instead of quitting, matching the Python app
	})

	// Auto-mount needs the event loop actually running (dialogs/UI updates
	// aren't safe before that), so it's wired to the "app started" hook
	// rather than called directly here.
	fyneApp.Lifecycle().SetOnStarted(func() {
		rm.autoMountAll()
		rm.checkForUpdate(false)
	})

	if rm.cfg.StartMinimized {
		// Deliberately skip win.Show(): ShowAndRun() would force it open
		// regardless, which is why this didn't work before. The tray icon
		// (already wired in setupTray) keeps the app reachable.
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

type rcloneManager struct {
	appDir string
	store  engine.Store
	cfg    engine.Config
	win    fyne.Window

	table         *widget.Table
	rcPathEntry   *widget.Entry
	rcVersionText *widget.Label

	activeMu sync.Mutex
	active   map[string]*runningMount
}

func (rm *rcloneManager) build() {
	rm.buildTable()

	top := container.NewVBox(
		rm.buildHeaderRow(),
		rm.buildRclonePathRow(),
		rm.buildStartupOptionsRow(),
	)

	addBtn := widget.NewButtonWithIcon("추가", nil, func() { rm.showMountDialog(nil) })
	bottom := container.NewBorder(nil, nil, nil, addBtn)

	rm.win.SetContent(container.NewBorder(top, bottom, nil, nil, rm.table))
}

func (rm *rcloneManager) buildHeaderRow() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("RcloneManager", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	versionBadge := widget.NewButton("v"+appVersion, func() { rm.checkForUpdate(true) })
	issueBtn := widget.NewButtonWithIcon("!", nil, func() {
		if u, err := url.Parse(issueURL); err == nil {
			_ = fyne.CurrentApp().OpenURL(u)
		}
	})
	return container.NewBorder(nil, nil, container.NewHBox(title, versionBadge), issueBtn)
}

func (rm *rcloneManager) buildRclonePathRow() fyne.CanvasObject {
	rm.rcPathEntry = widget.NewEntry()
	rm.rcPathEntry.SetText(rm.cfg.RclonePath)
	rm.rcPathEntry.SetPlaceHolder("rclone.exe 경로")

	browseBtn := widget.NewButtonWithIcon("", nil, func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()
			rm.rcPathEntry.SetText(reader.URI().Path())
			rm.cfg.RclonePath = reader.URI().Path()
			rm.persist()
			rm.refreshVersionLabel()
		}, rm.win)
		fd.Show()
	})

	rm.rcPathEntry.OnSubmitted = func(s string) {
		rm.cfg.RclonePath = strings.TrimSpace(s)
		rm.persist()
		rm.refreshVersionLabel()
	}

	rm.rcVersionText = widget.NewLabel("rclone 확인 중...")

	return container.NewBorder(nil, nil, widget.NewLabel("rclone 경로:"),
		container.NewHBox(browseBtn, rm.rcVersionText), rm.rcPathEntry)
}

func (rm *rcloneManager) buildStartupOptionsRow() fyne.CanvasObject {
	autoStart := widget.NewCheck("시작 시 자동 실행", func(checked bool) {
		if err := engine.SetStartup(checked); err != nil {
			dialog.ShowError(err, rm.win)
		}
	})
	autoStart.SetChecked(engine.IsStartupEnabled())

	autoMount := widget.NewCheck("시작 시 자동 마운트", func(checked bool) {
		rm.cfg.AutoMount = checked
		rm.persist()
	})
	autoMount.SetChecked(rm.cfg.AutoMount)

	startMinimized := widget.NewCheck("시작 시 트레이로 최소화", func(checked bool) {
		rm.cfg.StartMinimized = checked
		rm.persist()
	})
	startMinimized.SetChecked(rm.cfg.StartMinimized)

	return container.NewHBox(autoStart, autoMount, startMinimized)
}

// ── mount table ──

func (rm *rcloneManager) buildTable() {
	rm.table = widget.NewTable(
		func() (int, int) { return len(rm.cfg.Mounts), colCount },
		func() fyne.CanvasObject { return container.NewStack() },
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			rm.updateTableCell(id, cell.(*fyne.Container))
		},
	)
	rm.table.ShowHeaderRow = true
	rm.table.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}
	rm.table.UpdateHeader = func(id widget.TableCellID, o fyne.CanvasObject) {
		headers := [colCount]string{"자동", "드라이브", "리모트(서브경로)", "상태", ""}
		o.(*widget.Label).SetText(headers[id.Col])
	}
	rm.table.SetColumnWidth(colAuto, 50)
	rm.table.SetColumnWidth(colDrive, 80)
	rm.table.SetColumnWidth(colRemote, 280)
	rm.table.SetColumnWidth(colStatus, 80)
	rm.table.SetColumnWidth(colActions, 200)
}

// updateTableCell fills in one cell. CreateCell can't know in advance
// which column a recycled template will be asked to render, so each
// helper (cellCheck/cellLabel/cellActionButtons) replaces the cell's
// content if it isn't already the right widget type.
func (rm *rcloneManager) updateTableCell(id widget.TableCellID, cell *fyne.Container) {
	if id.Row >= len(rm.cfg.Mounts) {
		return
	}
	m := rm.cfg.Mounts[id.Row]

	switch id.Col {
	case colAuto:
		check := rm.cellCheck(cell)
		check.SetChecked(m.AutoMount)
		check.OnChanged = func(checked bool) {
			m.AutoMount = checked
			rm.saveMount(m)
		}

	case colDrive:
		label := rm.cellLabel(cell)
		drive := strings.TrimSpace(m.Drive)
		if drive == "" {
			drive = "(자동)"
		}
		label.SetText(drive)

	case colRemote:
		label := rm.cellLabel(cell)
		label.SetText(fmt.Sprintf("%s:%s", m.Remote, m.RemotePath))

	case colStatus:
		label := rm.cellLabel(cell)
		if rm.isRunning(m.ID) {
			label.SetText("연결됨")
		} else {
			label.SetText("해제됨")
		}

	case colActions:
		toggle, editBtn, delBtn := rm.cellActionButtons(cell)
		running := rm.isRunning(m.ID)
		if running {
			toggle.SetText("해제")
		} else {
			toggle.SetText("마운트")
		}
		toggle.OnTapped = func() {
			if running {
				rm.unmount(m.ID)
			} else {
				rm.mount(m)
			}
		}
		editBtn.OnTapped = func() { rm.showMountDialog(&m) }
		delBtn.OnTapped = func() { rm.confirmDelete(m) }
	}
}

func (rm *rcloneManager) isRunning(mountID string) bool {
	rm.activeMu.Lock()
	defer rm.activeMu.Unlock()
	_, running := rm.active[mountID]
	return running
}

func (rm *rcloneManager) cellCheck(cell *fyne.Container) *widget.Check {
	if len(cell.Objects) == 1 {
		if c, ok := cell.Objects[0].(*widget.Check); ok {
			return c
		}
	}
	c := widget.NewCheck("", nil)
	cell.Objects = []fyne.CanvasObject{c}
	return c
}

func (rm *rcloneManager) cellLabel(cell *fyne.Container) *widget.Label {
	if len(cell.Objects) == 1 {
		if l, ok := cell.Objects[0].(*widget.Label); ok {
			return l
		}
	}
	l := widget.NewLabel("")
	cell.Objects = []fyne.CanvasObject{l}
	return l
}

func (rm *rcloneManager) cellActionButtons(cell *fyne.Container) (toggle, edit, del *widget.Button) {
	if len(cell.Objects) == 1 {
		if row, ok := cell.Objects[0].(*fyne.Container); ok && len(row.Objects) == 3 {
			return row.Objects[0].(*widget.Button), row.Objects[1].(*widget.Button), row.Objects[2].(*widget.Button)
		}
	}
	toggle = widget.NewButton("", nil)
	edit = widget.NewButton("편집", nil)
	del = widget.NewButton("삭제", nil)
	cell.Objects = []fyne.CanvasObject{container.NewHBox(toggle, edit, del)}
	return toggle, edit, del
}

// ── add / edit dialog ──

func (rm *rcloneManager) showMountDialog(existing *engine.Mount) {
	remoteEntry := widget.NewEntry()
	pathEntry := widget.NewEntry()
	driveEntry := widget.NewEntry()
	driveEntry.SetPlaceHolder("예: Z: (비우면 자동)")
	cacheDirEntry := widget.NewEntry()
	cacheModeSelect := widget.NewSelect([]string{"", "off", "minimal", "writes", "full"}, nil)
	extraFlagsEntry := widget.NewMultiLineEntry()
	extraFlagsEntry.SetPlaceHolder("--flag=value;--flag2 value2")

	if existing != nil {
		remoteEntry.SetText(existing.Remote)
		pathEntry.SetText(existing.RemotePath)
		driveEntry.SetText(existing.Drive)
		cacheDirEntry.SetText(existing.CacheDir)
		cacheModeSelect.SetSelected(existing.CacheMode)
		extraFlagsEntry.SetText(existing.ExtraFlags)
	}

	form := dialog.NewForm(
		formTitle(existing), "저장", "취소",
		[]*widget.FormItem{
			widget.NewFormItem("리모트 이름", remoteEntry),
			widget.NewFormItem("서브 디렉토리", pathEntry),
			widget.NewFormItem("드라이브 문자", driveEntry),
			widget.NewFormItem("캐시 디렉토리", cacheDirEntry),
			widget.NewFormItem("캐시 모드", cacheModeSelect),
			widget.NewFormItem("추가 플래그", extraFlagsEntry),
		},
		func(ok bool) {
			if !ok {
				return
			}
			if strings.TrimSpace(remoteEntry.Text) == "" {
				dialog.ShowInformation("알림", "리모트 이름을 입력해 주세요.", rm.win)
				return
			}

			m := engine.Mount{
				ID:         idOrNew(existing),
				Remote:     strings.TrimSpace(remoteEntry.Text),
				RemotePath: strings.TrimSpace(pathEntry.Text),
				Drive:      strings.TrimSpace(driveEntry.Text),
				CacheDir:   strings.TrimSpace(cacheDirEntry.Text),
				CacheMode:  cacheModeSelect.Selected,
				ExtraFlags: engine.NormalizeFlags(extraFlagsEntry.Text),
				AutoMount:  existing != nil && existing.AutoMount,
			}
			rm.saveMount(m)
		},
		rm.win,
	)
	form.Resize(fyne.NewSize(420, 360))
	form.Show()
}

func formTitle(existing *engine.Mount) string {
	if existing == nil {
		return "마운트 추가"
	}
	return "마운트 편집"
}

func idOrNew(existing *engine.Mount) string {
	if existing != nil {
		return existing.ID
	}
	return engine.NewMountID()
}

func (rm *rcloneManager) saveMount(m engine.Mount) {
	found := false
	for i, existing := range rm.cfg.Mounts {
		if existing.ID == m.ID {
			rm.cfg.Mounts[i] = m
			found = true
			break
		}
	}
	if !found {
		rm.cfg.Mounts = append(rm.cfg.Mounts, m)
	}
	rm.persist()
}

func (rm *rcloneManager) confirmDelete(m engine.Mount) {
	dialog.ShowConfirm("삭제", fmt.Sprintf("%s:%s 마운트 설정을 삭제할까요?", m.Remote, m.RemotePath),
		func(ok bool) {
			if !ok {
				return
			}
			rm.unmount(m.ID)
			kept := rm.cfg.Mounts[:0]
			for _, existing := range rm.cfg.Mounts {
				if existing.ID != m.ID {
					kept = append(kept, existing)
				}
			}
			rm.cfg.Mounts = kept
			rm.persist()
		}, rm.win)
}

func (rm *rcloneManager) persist() {
	if err := rm.store.Save(rm.cfg); err != nil {
		dialog.ShowError(err, rm.win)
		return
	}
	rm.table.Refresh()
}

// ── mounting ──

func (rm *rcloneManager) rcloneExePath() (string, bool) {
	p := strings.TrimSpace(rm.cfg.RclonePath)
	if p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	fallback := filepath.Join(rm.appDir, "rclone.exe")
	if _, err := os.Stat(fallback); err == nil {
		return fallback, true
	}
	return "", false
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

// startNetworkMonitor polls connectivity every 10s and reacts only on a
// state *transition* (disconnected->connected or vice versa) — mirrors the
// Python version's _start_net_monitor exactly, including starting from an
// "unknown" state so the very first check always fires once (connected at
// startup re-triggers auto-mount as a safety net; disconnected at startup
// does nothing since there's nothing mounted yet to tear down).
func (rm *rcloneManager) startNetworkMonitor() {
	go func() {
		var wasConnected *bool
		for {
			connected := engine.IsInternetAvailable("8.8.8.8", 53, 3*time.Second)

			if wasConnected == nil || *wasConnected != connected {
				c := connected
				wasConnected = &c
				if connected {
					fyne.Do(func() { rm.autoMountAll() })
				} else {
					fyne.Do(func() { rm.unmountAllOnDisconnect() })
				}
			}

			time.Sleep(10 * time.Second)
		}
	}()
}

func (rm *rcloneManager) mount(m engine.Mount) {
	if rm.isRunning(m.ID) {
		return // already mounted — avoid spawning a duplicate rclone process
	}

	exe, ok := rm.rcloneExePath()
	if !ok {
		dialog.ShowInformation("알림", "rclone.exe를 찾을 수 없습니다. 먼저 rclone 경로를 등록해 주세요.", rm.win)
		return
	}

	args := engine.BuildCmd(exe, m)
	cmd := exec.Command(args[0], args[1:]...)
	engine.ConfigureBackgroundProcess(cmd) // hide console window, own process group
	if err := cmd.Start(); err != nil {
		dialog.ShowError(err, rm.win)
		return
	}

	done := make(chan struct{})
	rm.activeMu.Lock()
	rm.active[m.ID] = &runningMount{cmd: cmd, done: done}
	rm.activeMu.Unlock()
	rm.table.Refresh()

	go func() {
		_ = cmd.Wait() // process exits when unmounted (or on error)
		close(done)
		rm.activeMu.Lock()
		delete(rm.active, m.ID)
		rm.activeMu.Unlock()
		fyne.Do(func() { rm.table.Refresh() })
	}()
}

// unmount asks a running mount to stop gracefully (so rclone unmounts its
// WinFsp filesystem cleanly instead of leaving the drive letter stuck),
// falling back to a hard Kill() if it doesn't exit in time. Runs in a
// goroutine since it can block for a few seconds and this is called from a
// button handler on the UI thread.
func (rm *rcloneManager) unmount(mountID string) {
	rm.activeMu.Lock()
	running, ok := rm.active[mountID]
	rm.activeMu.Unlock()
	if !ok || running.cmd.Process == nil {
		return
	}

	go func() {
		if err := engine.SignalGracefulStop(running.cmd.Process.Pid); err != nil {
			_ = running.cmd.Process.Kill()
			return
		}
		select {
		case <-running.done:
			// exited cleanly on its own
		case <-time.After(5 * time.Second):
			_ = running.cmd.Process.Kill()
		}
	}()
}

// ── rclone version label ──

func (rm *rcloneManager) refreshVersionLabel() {
	exe, ok := rm.rcloneExePath()
	if !ok {
		rm.rcVersionText.SetText("rclone 다운로드 필요")
		return
	}
	go func() {
		cmd := exec.Command(exe, "version")
		engine.ConfigureBackgroundProcess(cmd)
		out, err := cmd.CombinedOutput()
		text := "v알 수 없음"
		if err == nil {
			if ver, found := engine.ParseLocalRcloneVersion(string(out)); found {
				text = "rclone v" + ver
			}
		}
		fyne.Do(func() { rm.rcVersionText.SetText(text) })
	}()
}

// ── self-update ──

// checkForUpdate looks up the latest GitHub release and, if it's newer
// than appVersion, offers to install it. When manual is false (the
// silent startup check), nothing is shown unless an update is found —
// mirrors the Python version's quiet periodic check.
func (rm *rcloneManager) checkForUpdate(manual bool) {
	go func() {
		rel, err := engine.FetchLatestRelease(nil, engine.AppReleaseAPI)
		if err != nil {
			if manual {
				fyne.Do(func() { dialog.ShowError(err, rm.win) })
			}
			return
		}
		if engine.CompareVersions(appVersion, rel.Version) >= 0 {
			if manual {
				fyne.Do(func() { dialog.ShowInformation("업데이트 확인", "이미 최신 버전입니다.", rm.win) })
			}
			return
		}

		var assetURL string
		for _, a := range rel.Assets {
			if a.Name == "RcloneManager.zip" {
				assetURL = a.DownloadURL
				break
			}
		}
		if assetURL == "" {
			return // release published without the expected asset — nothing to offer
		}

		fyne.Do(func() {
			dialog.ShowConfirm("업데이트 가능",
				fmt.Sprintf("새 버전 v%s가 있습니다. 지금 업데이트할까요?\n(적용 후 앱이 자동으로 재시작됩니다)", rel.Version),
				func(ok bool) {
					if ok {
						rm.performUpdate(assetURL)
					}
				}, rm.win)
		})
	}()
}

// performUpdate downloads the release zip, swaps it into place, and
// relaunches — then quits this (now-outdated) process.
func (rm *rcloneManager) performUpdate(assetURL string) {
	progress := dialog.NewCustomWithoutButtons("업데이트 중",
		widget.NewLabel("새 버전을 다운로드하고 있습니다..."), rm.win)
	progress.Show()

	go func() {
		newExe, err := engine.DownloadAppUpdate(nil, rm.appDir, assetURL)
		if err != nil {
			fyne.Do(func() { progress.Hide(); dialog.ShowError(err, rm.win) })
			return
		}
		currentExe, err := os.Executable()
		if err != nil {
			fyne.Do(func() { progress.Hide(); dialog.ShowError(err, rm.win) })
			return
		}
		if err := engine.ApplyUpdate(currentExe, newExe); err != nil {
			fyne.Do(func() { progress.Hide(); dialog.ShowError(err, rm.win) })
			return
		}
		fyne.Do(func() {
			progress.Hide()
			fyne.CurrentApp().Quit() // the new version is already launching
		})
	}()
}

// ── tray ──

func (rm *rcloneManager) setupTray(fyneApp fyne.App) {
	desk, ok := fyneApp.(desktop.App)
	if !ok {
		return // no system tray support on this platform/build
	}
	// SetSystemTrayWindow gives left-click = show/hide the window (like a
	// normal Windows tray icon), right-click = the menu below. Fyne also
	// auto-appends its own "Quit" item to any tray menu, so we only add
	// "열기" here — adding our own quit item would just duplicate it.
	desk.SetSystemTrayWindow(rm.win)
	menu := fyne.NewMenu("RcloneManager",
		fyne.NewMenuItem("열기", func() { rm.win.Show() }),
	)
	desk.SetSystemTrayMenu(menu)
}
