package main

import (
	"testing"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func TestTrayShortLabel(t *testing.T) {
	Scenario(t, "GIVEN 드라이브 문자가 지정된 마운트 WHEN 짧은 라벨 결정 THEN 드라이브 문자를 쓴다", func(t *testing.T) {
		got := trayShortLabel(engine.Mount{Remote: "gds", Drive: "G:"})
		if got != "G:" {
			t.Errorf("got %q, 기대값 %q", got, "G:")
		}
	})
	Scenario(t, "GIVEN 드라이브 문자가 없는 마운트 WHEN 짧은 라벨 결정 THEN 리모트 이름으로 대체한다", func(t *testing.T) {
		got := trayShortLabel(engine.Mount{Remote: "gds"})
		if got != "gds" {
			t.Errorf("got %q, 기대값 %q (리모트 이름 폴백)", got, "gds")
		}
	})
}

func TestTrayDisplayLabel(t *testing.T) {
	m := engine.Mount{Remote: "PLEX", RemotePath: "KODI", Drive: "E:"}

	Scenario(t, "GIVEN 마운트가 실행 중임 WHEN 트레이 메뉴 항목 텍스트 구성 THEN ■ 아이콘과 함께 표시된다", func(t *testing.T) {
		got := trayDisplayLabel(m, true)
		if got != "■  E:  (PLEX:KODI)" {
			t.Errorf("got %q, 기대값 %q", got, "■  E:  (PLEX:KODI)")
		}
	})

	Scenario(t, "GIVEN 마운트가 중지 상태임 WHEN 트레이 메뉴 항목 텍스트 구성 THEN ▶ 아이콘과 함께 표시된다", func(t *testing.T) {
		got := trayDisplayLabel(m, false)
		if got != "▶  E:  (PLEX:KODI)" {
			t.Errorf("got %q, 기대값 %q", got, "▶  E:  (PLEX:KODI)")
		}
	})

	// 부정/경계 케이스: 서브경로가 없는 리모트(예: "nas:")는 괄호 안에
	// 콜론만 덩그러니 남으면 안 된다.
	Scenario(t, "GIVEN 서브경로가 없는 리모트(예: nas:) WHEN 트레이 메뉴 항목 텍스트 구성 THEN 괄호 안에 불필요한 콜론이 남지 않는다 (경계 케이스)", func(t *testing.T) {
		bare := engine.Mount{Remote: "nas"}
		got := trayDisplayLabel(bare, false)
		if got != "▶  nas  (nas)" {
			t.Errorf("got %q, 기대값 %q", got, "▶  nas  (nas)")
		}
	})
}
