package main

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
)

// windowsFontTheme wraps another theme, swapping only the font — the rest
// (colors, sizes, icons) are delegated unchanged.
//
// This does NOT make Fyne render text the way native Windows apps do
// (Fyne rasterizes glyphs itself, without Windows' ClearType subpixel
// antialiasing — a rendering-pipeline difference no font choice can
// close). What it DOES fix is that Fyne's bundled default font isn't the
// one Windows actually uses for Korean UI text, so shapes/spacing looked
// subtly "off" next to native apps. Using 맑은 고딕 (Windows' own default
// Korean UI font since Vista) matches that.
type windowsFontTheme struct {
	fyne.Theme
	regular, bold fyne.Resource
}

// newWindowsFontTheme loads 맑은 고딕 from the Windows Fonts folder, if
// present. Falls back to base's own font entirely if the file can't be
// read (e.g. running on a non-Windows OS, or an unusual Windows install
// missing this font) — never errors, just silently keeps the default.
func newWindowsFontTheme(base fyne.Theme) fyne.Theme {
	t := &windowsFontTheme{Theme: base}

	winDir := os.Getenv("WINDIR")
	if winDir == "" {
		winDir = `C:\Windows`
	}
	fontsDir := filepath.Join(winDir, "Fonts")

	if data, err := os.ReadFile(filepath.Join(fontsDir, "malgun.ttf")); err == nil {
		t.regular = fyne.NewStaticResource("malgun.ttf", data)
	}
	if data, err := os.ReadFile(filepath.Join(fontsDir, "malgunbd.ttf")); err == nil {
		t.bold = fyne.NewStaticResource("malgunbd.ttf", data)
	}
	return t
}

func (t *windowsFontTheme) Font(style fyne.TextStyle) fyne.Resource {
	// 모노스페이스(로그/코드 느낌)는 원래 테마의 고정폭 폰트를 그대로 쓴다 —
	// 맑은 고딕은 고정폭 폰트가 아니다.
	if style.Monospace {
		return t.Theme.Font(style)
	}
	if style.Bold && t.bold != nil {
		return t.bold
	}
	if t.regular != nil {
		return t.regular
	}
	return t.Theme.Font(style)
}
