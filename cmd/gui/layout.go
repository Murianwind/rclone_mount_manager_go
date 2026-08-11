package main

import (
	"image/color"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func (rm *rcloneManager) build() {
	rm.buildTable()

	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(0, 16))

	// Table의 MinSize()는 컬럼 폭 합계를 반영하지 않아서, 이게 없으면
	// 기본 창 너비 계산이 표 내용보다 좁게 잡힌다. 눈에는 안 보이지만
	// 폭만 차지하는 사각형으로 "창이 이 정도는 돼야 자연스럽다"는 크기
	// 힌트를 준다. (주의: Fyne 창은 이 힌트를 초기 크기 계산에는 쓰지만,
	// 사용자가 마우스로 드래그해 실제로 더 작게 줄이는 것까지 막아주진
	// 않는다 — 이 버전엔 그런 강제 최소 크기 기능이 없다.)
	minWidthSpacer := canvas.NewRectangle(color.Transparent)
	minWidthSpacer.SetMinSize(fyne.NewSize(tableContentWidth+20, 0))

	top := container.NewVBox(
		rm.buildHeaderRow(),
		rm.buildRclonePathRow(),
		rm.buildStartupOptionsRow(),
		spacer,
		minWidthSpacer,
	)

	minTableHeight := canvas.NewRectangle(color.Transparent)
	minTableHeight.SetMinSize(fyne.NewSize(0, 200))
	tableArea := container.NewStack(minTableHeight, rm.table)

	addBtn := widget.NewButtonWithIcon("추가", nil, func() { rm.showMountDialog(nil, "") })
	upBtn := widget.NewButton("🔼", func() { rm.moveSelectedUp() })
	downBtn := widget.NewButton("🔽", func() { rm.moveSelectedDown() })
	importBtn := widget.NewButtonWithIcon("conf 가져오기", nil, func() { rm.showConfImportDialog() })
	bottom := container.NewBorder(nil, nil, nil,
		container.NewHBox(upBtn, downBtn, importBtn, addBtn))

	content := container.NewBorder(top, bottom, nil, nil, tableArea)

	// 표 바깥의 빈 공간(여백, 스페이서 자리 등)을 클릭하면 선택을
	// 해제한다 — 실제 위젯(버튼/입력창/표 셀)이 있는 자리는 그 위젯이
	// 클릭을 먼저 가져가므로 여기까지 안 온다.
	background := newTapCatcher(func() { rm.clearSelection() })
	rm.win.SetContent(container.NewStack(background, content))
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
	rm.rcPathEntry.MultiLine = true
	rm.rcPathEntry.Wrapping = fyne.TextWrapWord
	rm.rcPathEntry.SetMinRowsVisible(1) // 평소엔 한 줄만큼만 차지, 긴 경로는 줄바꿈+스크롤로 처리

	browseBtn := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
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
		fd.Resize(fyne.NewSize(700, 460)) // Show() 이후에 불러야 안전함 (이전엔 순서가 반대라 크래시 원인이었음)
	})

	rm.rcPathEntry.OnSubmitted = func(s string) {
		rm.cfg.RclonePath = strings.TrimSpace(s)
		rm.persist()
		rm.refreshVersionLabel()
	}

	rm.rcVersionText = widget.NewButton("rclone 확인 중...", func() { rm.checkRcloneUpdate(true) })

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

// rcloneExePath resolves the rclone.exe to use: the explicitly configured
// path if it exists, else appDir/rclone.exe.
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

func (rm *rcloneManager) refreshVersionLabel() {
	exe, ok := rm.rcloneExePath()
	if !ok {
		rm.rcVersionText.SetText("rclone 다운로드 필요")
		return
	}
	go func() {
		text := detectLocalRcloneVersion(exe)
		fyne.Do(func() { rm.rcVersionText.SetText(text) })
	}()
}
