package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

func TestWindowsFontTheme(t *testing.T) {
	base := theme.DefaultTheme()
	wt := newWindowsFontTheme(base)

	Scenario(t, "GIVEN 맑은 고딕 파일을 못 찾는 환경(리눅스 등) WHEN 일반 텍스트 폰트 조회 THEN panic 없이 기본 테마의 폰트로 폴백한다 (부정 케이스)", func(t *testing.T) {
		style := fyne.TextStyle{}
		got := wt.Font(style)
		want := base.Font(style)
		if got != want {
			t.Errorf("폰트 파일이 없으면 기본 테마와 동일한 리소스를 반환해야 함")
		}
	})

	Scenario(t, "GIVEN 모노스페이스 스타일 WHEN 폰트 조회 THEN 맑은 고딕이 아니라 항상 기본 테마의 고정폭 폰트를 쓴다", func(t *testing.T) {
		style := fyne.TextStyle{Monospace: true}
		got := wt.Font(style)
		want := base.Font(style)
		if got != want {
			t.Errorf("모노스페이스는 항상 기본 테마 폰트를 그대로 써야 함")
		}
	})

	// 색상/크기/아이콘은 그대로 위임되는지 확인 — 폰트만 바꾸는 래퍼라는 계약.
	Scenario(t, "GIVEN 색상 조회 WHEN 커스텀 테마를 통해 조회 THEN 기본 테마와 동일한 값을 그대로 위임한다", func(t *testing.T) {
		name := theme.ColorNameForeground
		if wt.Color(name, theme.VariantDark) != base.Color(name, theme.VariantDark) {
			t.Errorf("색상은 그대로 위임돼야 하는데 다름")
		}
	})
}
