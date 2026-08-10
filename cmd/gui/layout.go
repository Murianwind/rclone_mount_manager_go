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
	// 창을 표 내용보다 좁게 줄일 수 있어 액션 컬럼이 잘려 보인다. 눈에는
	// 안 보이지만 폭만 차지하는 사각형으로 창의 최소 너비를 강제한다.
	// (Border의 최소 너비는 각 구역 중 최댓값이라 top 안에 둬도 안전하다 —
	// 반면 높이는 top의 자식들 높이가 그대로 더해지므로 최소 높이는
	// 아래에서 표와 함께 별도로 강제한다.)
	minWidthSpacer := canvas.NewRectangle(color.Transparent)
	minWidthSpacer.SetMinSize(fyne.NewSize(tableContentWidth+20, 0))

	top := container.NewVBox(
		rm.buildHeaderRow(),
		rm.buildRclonePathRow(),
		rm.buildStartupOptionsRow(),
		spacer,
		minWidthSpacer,
	)

	// 표 영역 자체의 최소 높이를 강제한다 — 이게 없으면 세로로 줄일 때
	// 표가 거의 안 보이는 높이까지 눌린다. 표와 같은 자리에(Stack으로)
	// 겹쳐둬서 top의 높이에는 더해지지 않고 표 영역에만 반영되게 한다.
	// 저해상도 화면도 부담 없이 맞도록 값은 넉넉하지 않게(행 몇 개 볼
	// 정도로만) 잡았다.
	minTableHeight := canvas.NewRectangle(color.Transparent)
	minTableHeight.SetMinSize(fyne.NewSize(0, 200))
	tableArea := container.NewStack(minTableHeight, rm.table)

	addBtn := widget.NewButtonWithIcon("추가", nil, func() { rm.showMountDialog(nil, "") })
	upBtn := widget.NewButton("🔼", func() { rm.moveSelectedUp() })
	downBtn := widget.NewButton("🔽", func() { rm.moveSelectedDown() })
	importBtn := widget.NewButtonWithIcon("conf 가져오기", nil, func() { rm.showConfImportDialog() })
	bottom := container.NewBorder(nil, nil, nil,
		container.NewHBox(upBtn, downBtn, importBtn, addBtn))

	rm.win.SetContent(container.NewBorder(top, bottom, nil, nil, tableArea))
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
