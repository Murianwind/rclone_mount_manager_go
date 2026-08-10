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
	win.Resize(fyne.NewSize(720, 480))

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

	win.SetCloseIntercept(func() {
		win.Hide() // minimize to tray instead of quitting, matching the Python app
	})

	if rm.cfg.StartMinimized {
		win.Hide()
	}

	win.ShowAndRun()
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

	list          *widget.List
	rcPathEntry   *widget.Entry
	rcVersionText *widget.Label

	activeMu sync.Mutex
	active   map[string]*runningMount
}

func (rm *rcloneManager) build() {
	rm.list = widget.NewList(
		func() int { return len(rm.cfg.Mounts) },
		func() fyne.CanvasObject { return rm.newMountRow() },
		func(i widget.ListItemID, obj fyne.CanvasObject) { rm.updateMountRow(i, obj) },
	)

	top := container.NewVBox(
		rm.buildHeaderRow(),
		rm.buildRclonePathRow(),
		rm.buildStartupOptionsRow(),
		rm.buildTableHeaderRow(),
	)

	addBtn := widget.NewButtonWithIcon("추가", nil, func() { rm.showMountDialog(nil) })
	bottom := container.NewBorder(nil, nil, nil, addBtn)

	rm.win.SetContent(container.NewBorder(top, bottom, nil, nil, rm.list))
}

func (rm *rcloneManager) buildHeaderRow() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("RcloneManager", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	versionBadge := widget.NewLabel("v" + appVersion)
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

func (rm *rcloneManager) buildTableHeaderRow() fyne.CanvasObject {
	bold := func(s string) *widget.Label {
		return widget.NewLabelWithStyle(s, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}
	return container.NewGridWithColumns(5,
		bold("자동"), bold("드라이브"), bold("리모트(서브경로)"), bold("상태"), bold(""))
}

// ── mount list rows ──

// newMountRow builds one (empty, template) list row. updateMountRow fills
// it in by reaching into row.Objects — the columns here must match
// buildTableHeaderRow: 자동 | 드라이브 | 리모트(서브경로) | 상태 | 버튼.
func (rm *rcloneManager) newMountRow() fyne.CanvasObject {
	auto := widget.NewCheck("", nil)
	drive := widget.NewLabel("")
	remote := widget.NewLabel("")
	status := widget.NewLabel("")
	toggle := widget.NewButton("", nil)
	edit := widget.NewButton("편집", nil)
	del := widget.NewButton("삭제", nil)

	return container.NewGridWithColumns(5,
		auto, drive, remote, status, container.NewHBox(toggle, edit, del))
}

func (rm *rcloneManager) updateMountRow(i widget.ListItemID, obj fyne.CanvasObject) {
	if i >= len(rm.cfg.Mounts) {
		return
	}
	m := rm.cfg.Mounts[i]
	grid := obj.(*fyne.Container)

	auto := grid.Objects[0].(*widget.Check)
	drive := grid.Objects[1].(*widget.Label)
	remote := grid.Objects[2].(*widget.Label)
	status := grid.Objects[3].(*widget.Label)
	buttons := grid.Objects[4].(*fyne.Container)
	toggle := buttons.Objects[0].(*widget.Button)
	editBtn := buttons.Objects[1].(*widget.Button)
	delBtn := buttons.Objects[2].(*widget.Button)

	auto.SetChecked(m.AutoMount)
	auto.OnChanged = func(checked bool) {
		m.AutoMount = checked
		rm.saveMount(m)
	}

	driveText := strings.TrimSpace(m.Drive)
	if driveText == "" {
		driveText = "(자동)"
	}
	drive.SetText(driveText)
	remote.SetText(fmt.Sprintf("%s:%s", m.Remote, m.RemotePath))

	rm.activeMu.Lock()
	_, running := rm.active[m.ID]
	rm.activeMu.Unlock()

	if running {
		status.SetText("연결됨")
		toggle.SetText("해제")
	} else {
		status.SetText("해제됨")
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
	rm.list.Refresh()
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

func (rm *rcloneManager) mount(m engine.Mount) {
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
	rm.list.Refresh()

	go func() {
		_ = cmd.Wait() // process exits when unmounted (or on error)
		close(done)
		rm.activeMu.Lock()
		delete(rm.active, m.ID)
		rm.activeMu.Unlock()
		fyne.Do(func() { rm.list.Refresh() })
	}()
}

// unmount asks a running mount to stop gracefully (so rclone unmounts its
// WinFsp filesystem cleanly instead of leaving the drive letter stuck),
// falling back to a hard Kill() if it doesn't exit in time. Runs the wait
// in a goroutine since it can block for a few seconds and this is called
// from a button handler on the UI thread.
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

// ── tray ──

func (rm *rcloneManager) setupTray(fyneApp fyne.App) {
	desk, ok := fyneApp.(desktop.App)
	if !ok {
		return // no system tray support on this platform/build
	}
	// Note: Fyne's tray API only supports a menu (shown on any click) —
	// there's no separate "left-click restores the window" action like a
	// typical Windows tray icon. "열기" is kept as the first/default item
	// so restoring the window is always one click away.
	menu := fyne.NewMenu("RcloneManager",
		fyne.NewMenuItem("열기", func() { rm.win.Show() }),
		fyne.NewMenuItem("종료", func() { fyneApp.Quit() }),
	)
	desk.SetSystemTrayMenu(menu)
}
