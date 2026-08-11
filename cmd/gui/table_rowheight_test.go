package main

import (
	"image"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/software"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

// TestRenderedTable_RowsAreVerticallySpread is a direct regression test
// for a real bug: Table auto-detects its row height from CreateCell's
// *empty* template object's MinSize(). When that template was a bare,
// empty container.NewStack(), its MinSize was 0×0, so Table gave every
// row zero height and all rows rendered stacked on top of each other
// instead of one below another. Rendering the actual window and checking
// that content reaches well below the header — where only a later row
// could legitimately draw — catches this class of bug even though
// Table's row height isn't otherwise inspectable from outside the widget.
func TestRenderedTable_RowsAreVerticallySpread(t *testing.T) {
	app := fynetest.NewApp()
	dir := t.TempDir()
	win := app.NewWindow("test")

	rm := newRcloneManager(dir, engine.RotatingLog{Path: filepath.Join(dir, "x.log"), MaxLines: 10}, win)
	rm.cfg.Mounts = []engine.Mount{
		{ID: "1", Remote: "PLEX", RemotePath: "KODI", Drive: "E:"},
		{ID: "2", Remote: "gds", RemotePath: "GDRIVE/VIDEO", Drive: "G:"},
		{ID: "3", Remote: "ondrive", RemotePath: "Webtoon", Drive: "H:"},
		{ID: "4", Remote: "PikPak", RemotePath: "", Drive: "I:"},
	}
	rm.build()
	win.Resize(fyne.NewSize(820, 520))

	img := software.RenderCanvas(win.Canvas(), theme.DefaultTheme())

	Scenario(t, "GIVEN 표에 마운트 4개가 있음 WHEN 창을 렌더링 THEN 헤더 훨씬 아래(4번째 행 자리)까지 실제 내용이 그려진다 (회귀 테스트)", func(t *testing.T) {
		// 4개 행이 제대로 세로로 펼쳐졌다면 y=300 부근에 마지막 행의
		// 텍스트/버튼이 있어야 한다. 버그가 있으면 모든 행이 헤더 바로
		// 아래(y=200 이하)에 겹쳐 그려지고 그 밑은 전부 배경색뿐이다.
		if !hasNonBackgroundPixelInRow(img, 300) {
			t.Fatalf("y=300 부근에 내용이 전혀 없음 — 행들이 겹쳐서 위쪽에만 뭉쳐 그려지는 버그가 재발한 것으로 보임")
		}
	})
}

// hasNonBackgroundPixelInRow scans a horizontal strip around y and reports
// whether any pixel differs meaningfully from the image's own background
// color (sampled from a corner, since Fyne dark/light theme can vary).
func hasNonBackgroundPixelInRow(img image.Image, y int) bool {
	bounds := img.Bounds()
	if y >= bounds.Dy() {
		return false
	}
	bg := img.At(bounds.Min.X, bounds.Min.Y)
	bgR, bgG, bgB, _ := bg.RGBA()

	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		r, g, b, _ := img.At(x, bounds.Min.Y+y).RGBA()
		diff := absInt32(int32(r)-int32(bgR)) + absInt32(int32(g)-int32(bgG)) + absInt32(int32(b)-int32(bgB))
		if diff > 6000 { // 눈에 띄는 색 차이만 "내용 있음"으로 침
			return true
		}
	}
	return false
}

func absInt32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
