package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

// showMountDialog opens the add/edit form. existing == nil means "add a
// new mount"; prefillRemote pre-fills the remote-name field for that case
// (used by the "가져오기" action on a raw remote row) and is ignored when
// existing != nil.
func (rm *rcloneManager) showMountDialog(existing *engine.Mount, prefillRemote string) {
	remoteEntry := widget.NewEntry()
	pathEntry := widget.NewEntry()
	wrapEntry(pathEntry) // 서브 디렉토리 경로는 길어질 수 있음
	driveEntry := widget.NewEntry()
	driveEntry.SetPlaceHolder("예: Z: (비우면 자동)")
	cacheDirEntry := widget.NewEntry()
	wrapEntry(cacheDirEntry) // 캐시 디렉토리 경로도 길어질 수 있음
	cacheModeSelect := widget.NewSelect([]string{"", "off", "minimal", "writes", "full"}, nil)
	extraFlagsEntry := widget.NewMultiLineEntry()
	extraFlagsEntry.SetPlaceHolder("--flag=value;--flag2 value2")
	extraFlagsEntry.Wrapping = fyne.TextWrapWord

	if existing != nil {
		remoteEntry.SetText(existing.Remote)
		pathEntry.SetText(existing.RemotePath)
		driveEntry.SetText(existing.Drive)
		cacheDirEntry.SetText(existing.CacheDir)
		cacheModeSelect.SetSelected(existing.CacheMode)
		extraFlagsEntry.SetText(existing.ExtraFlags)
	} else if prefillRemote != "" {
		remoteEntry.SetText(prefillRemote)
	}

	form := dialog.NewForm(
		mountDialogTitle(existing != nil), "저장", "취소",
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
				ID:         mountIDFor(existing),
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
	form.Resize(fyne.NewSize(440, 420))
	form.Show()
}

// wrapEntry turns a single-line Entry into one that word-wraps and scrolls
// internally instead of clipping when its content is longer than the
// field is wide — used for path-shaped fields (서브 디렉토리, 캐시
// 디렉토리) where long values are common.
func wrapEntry(e *widget.Entry) {
	e.MultiLine = true
	e.Wrapping = fyne.TextWrapWord
	e.SetMinRowsVisible(1)
}

func mountDialogTitle(editing bool) string {
	if editing {
		return "마운트 편집"
	}
	return "마운트 추가"
}

// mountIDFor keeps an existing mount's ID on edit, or mints a new one when
// adding.
func mountIDFor(existing *engine.Mount) string {
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

func (rm *rcloneManager) confirmDeleteRemote(r engine.Remote) {
	dialog.ShowConfirm("삭제", fmt.Sprintf("원본 '%s'을 목록에서 삭제할까요? (rclone.conf 자체는 건드리지 않습니다)", r.Name),
		func(ok bool) {
			if !ok {
				return
			}
			kept := rm.cfg.Remotes[:0]
			for _, existing := range rm.cfg.Remotes {
				if existing.Name != r.Name {
					kept = append(kept, existing)
				}
			}
			rm.cfg.Remotes = kept
			rm.persist()
		}, rm.win)
}

// showMountFailureDialog shows rclone's own error output (its stderr) so
// the user can see *why* a mount failed, instead of it just silently going
// back to "해제됨". Also points at the log file for the full history.
func (rm *rcloneManager) showMountFailureDialog(m engine.Mount, detail string) {
	label := widget.NewLabel(mountFailureMessage(m, detail, rm.log.Path))
	label.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(label)
	scroll.SetMinSize(fyne.NewSize(420, 220))
	dialog.ShowCustom("마운트 오류", "확인", scroll, rm.win)
}

// mountFailureMessage builds the failure-dialog text. Pulled out as a pure
// function so the formatting can be tested without a running Fyne app.
func mountFailureMessage(m engine.Mount, detail, logPath string) string {
	if strings.TrimSpace(detail) == "" {
		detail = "(rclone에서 별도 오류 메시지를 출력하지 않았습니다)"
	}
	return fmt.Sprintf(
		"%s:%s 마운트에 실패했습니다.\n\nrclone 오류:\n%s\n\n자세한 내용은 로그 파일을 확인하세요:\n%s",
		m.Remote, m.RemotePath, detail, logPath,
	)
}
