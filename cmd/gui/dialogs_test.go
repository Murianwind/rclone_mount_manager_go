package main

import (
	"strings"
	"testing"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func TestMountDialogTitle(t *testing.T) {
	Scenario(t, "GIVEN 새 마운트를 추가하는 상황 WHEN 다이얼로그 제목 결정 THEN '마운트 추가'가 된다", func(t *testing.T) {
		if got := mountDialogTitle(false); got != "마운트 추가" {
			t.Errorf("mountDialogTitle(false) = %q, 기대값 %q", got, "마운트 추가")
		}
	})
	Scenario(t, "GIVEN 기존 마운트를 편집하는 상황 WHEN 다이얼로그 제목 결정 THEN '마운트 편집'이 된다", func(t *testing.T) {
		if got := mountDialogTitle(true); got != "마운트 편집" {
			t.Errorf("mountDialogTitle(true) = %q, 기대값 %q", got, "마운트 편집")
		}
	})
}

func TestMountIDFor(t *testing.T) {
	Scenario(t, "GIVEN 기존 마운트를 편집하는 상황 WHEN ID 결정 THEN 기존 ID를 그대로 유지한다", func(t *testing.T) {
		existing := &engine.Mount{ID: "abc-123"}
		if got := mountIDFor(existing); got != "abc-123" {
			t.Errorf("mountIDFor(existing) = %q, 기존 ID(abc-123)가 유지돼야 함", got)
		}
	})
	Scenario(t, "GIVEN 새 마운트를 추가하는 상황(existing == nil) WHEN ID 결정 THEN 새 ID를 발급한다", func(t *testing.T) {
		if got := mountIDFor(nil); got == "" {
			t.Errorf("mountIDFor(nil)은 비어있지 않은 새 ID를 반환해야 함")
		}
	})
}

func TestMountFailureMessage(t *testing.T) {
	m := engine.Mount{Remote: "PLEX", RemotePath: "KODI"}

	Scenario(t, "GIVEN rclone의 실제 오류 내용이 있음 WHEN 실패 메시지 구성 THEN 리모트 경로·오류 상세·로그 경로가 전부 포함된다", func(t *testing.T) {
		msg := mountFailureMessage(m, "CRITICAL: Fatal error: failed to mount", "/tmp/RcloneManager.log")
		if !strings.Contains(msg, "PLEX:KODI") {
			t.Errorf("메시지에 remote:path가 포함돼야 함: %q", msg)
		}
		if !strings.Contains(msg, "CRITICAL: Fatal error") {
			t.Errorf("메시지에 rclone 오류 상세가 포함돼야 함: %q", msg)
		}
		if !strings.Contains(msg, "/tmp/RcloneManager.log") {
			t.Errorf("메시지에 로그 파일 경로가 포함돼야 함: %q", msg)
		}
	})

	// 부정 케이스: rclone이 stderr에 아무것도 안 남긴 경우에도 메시지가
	// 텅 비면 안 되고, 대체 안내 문구가 나와야 한다.
	Scenario(t, "GIVEN rclone이 오류 메시지를 전혀 출력하지 않음 WHEN 실패 메시지 구성 THEN 대체 안내 문구를 보여준다 (부정 케이스)", func(t *testing.T) {
		msg := mountFailureMessage(m, "", "/tmp/RcloneManager.log")
		if !strings.Contains(msg, "별도 오류 메시지를 출력하지 않았습니다") {
			t.Errorf("상세 내용이 없을 때의 대체 문구가 있어야 함: %q", msg)
		}
	})
}
