package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

// runningMount tracks a live rclone mount process.
type runningMount struct {
	cmd *exec.Cmd
}

func main() {
	appDir := mustAppDir()

	fyneApp := app.NewWithID("com.murianwind.rclonemanager")
	win := fyneApp.NewWindow("RcloneManager")
	win.Resize(fyne.NewSize(480, 420))

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

	win.ShowAndRun()
}

// mustAppDir returns the directory the running executable lives in — the
// Go equivalent of the Python version's APP_DIR (Path(sys.executable).parent
// when frozen). mounts.json, rclone.exe, and the log file all live here.
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

	addBtn := widget.NewButtonWithIcon("추가", nil, func() { rm.showMountDialog(nil) })

	rm.rcVersionText = widget.NewLabel("rclone 확인 중...")

	bottom := container.NewBorder(nil, nil, rm.rcVersionText, addBtn)
	rm.win.SetContent(container.NewBorder(nil, bottom, nil, nil, rm.list))
}

// ── mount list rows ──

// newMountRow builds one (empty, template) list row. updateMountRow fills
// it in by reaching into row.Objects — see the index comments there for
// the layout this must match: label | [status, toggle, edit, delete].
func (rm *rcloneManager) newMountRow() fyne.CanvasObject {
	label := widget.NewLabel("")
	status := widget.NewLabel("")
	toggle := widget.NewButton("", nil)
	edit := widget.NewButtonWithIcon("", nil, nil)
	del := widget.NewButtonWithIcon("", nil, nil)

	return container.NewBorder(nil, nil, nil,
		container.NewHBox(status, toggle, edit, del), label)
}

func (rm *rcloneManager) updateMountRow(i widget.ListItemID, obj fyne.CanvasObject) {
	if i >= len(rm.cfg.Mounts) {
		return
	}
	m := rm.cfg.Mounts[i]
	row := obj.(*fyne.Container)

	label := row.Objects[0].(*widget.Label)
	right := row.Objects[1].(*fyne.Container)
	status := right.Objects[0].(*widget.Label)
	toggle := right.Objects[1].(*widget.Button)
	editBtn := right.Objects[2].(*widget.Button)
	delBtn := right.Objects[3].(*widget.Button)

	drive := strings.TrimSpace(m.Drive)
	if drive == "" {
		drive = "(자동)"
	}
	label.SetText(fmt.Sprintf("%s:%s → %s", m.Remote, m.RemotePath, drive))

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
		rm.list.Refresh()
	}
	editBtn.SetIcon(nil)
	editBtn.SetText("편집")
	editBtn.OnTapped = func() { rm.showMountDialog(&m) }
	delBtn.SetText("삭제")
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
	if err := cmd.Start(); err != nil {
		dialog.ShowError(err, rm.win)
		return
	}

	rm.activeMu.Lock()
	rm.active[m.ID] = &runningMount{cmd: cmd}
	rm.activeMu.Unlock()

	go func() {
		_ = cmd.Wait() // process exits when unmounted (or on error)
		rm.activeMu.Lock()
		delete(rm.active, m.ID)
		rm.activeMu.Unlock()
		rm.list.Refresh()
	}()
}

func (rm *rcloneManager) unmount(mountID string) {
	rm.activeMu.Lock()
	running, ok := rm.active[mountID]
	rm.activeMu.Unlock()
	if !ok || running.cmd.Process == nil {
		return
	}
	_ = running.cmd.Process.Kill()
}

// ── rclone version label ──

func (rm *rcloneManager) refreshVersionLabel() {
	exe, ok := rm.rcloneExePath()
	if !ok {
		rm.rcVersionText.SetText("rclone 다운로드 필요")
		return
	}
	go func() {
		out, err := exec.Command(exe, "version").CombinedOutput()
		if err != nil {
			rm.rcVersionText.SetText("v알 수 없음")
			return
		}
		ver, found := engine.ParseLocalRcloneVersion(string(out))
		if !found {
			rm.rcVersionText.SetText("v알 수 없음")
			return
		}
		rm.rcVersionText.SetText("rclone v" + ver)
	}()
}

// ── tray ──

func (rm *rcloneManager) setupTray(fyneApp fyne.App) {
	desk, ok := fyneApp.(desktop.App)
	if !ok {
		return // no system tray support on this platform/build
	}
	menu := fyne.NewMenu("RcloneManager",
		fyne.NewMenuItem("열기", func() { rm.win.Show() }),
		fyne.NewMenuItem("종료", func() { fyneApp.Quit() }),
	)
	desk.SetSystemTrayMenu(menu)
}
