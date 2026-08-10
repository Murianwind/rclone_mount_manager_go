package main

import (
	"strings"
	"testing"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func TestMountDialogTitle(t *testing.T) {
	if got := mountDialogTitle(false); got != "마운트 추가" {
		t.Errorf("mountDialogTitle(false) = %q, want %q", got, "마운트 추가")
	}
	if got := mountDialogTitle(true); got != "마운트 편집" {
		t.Errorf("mountDialogTitle(true) = %q, want %q", got, "마운트 편집")
	}
}

func TestMountIDFor(t *testing.T) {
	existing := &engine.Mount{ID: "abc-123"}
	if got := mountIDFor(existing); got != "abc-123" {
		t.Errorf("mountIDFor(existing) = %q, want existing ID preserved", got)
	}
	if got := mountIDFor(nil); got == "" {
		t.Errorf("mountIDFor(nil) should mint a new non-empty ID")
	}
}

func TestMountFailureMessage(t *testing.T) {
	m := engine.Mount{Remote: "PLEX", RemotePath: "KODI"}

	withDetail := mountFailureMessage(m, "CRITICAL: Fatal error: failed to mount", "/tmp/RcloneManager.log")
	if !strings.Contains(withDetail, "PLEX:KODI") {
		t.Errorf("expected remote:path in message: %q", withDetail)
	}
	if !strings.Contains(withDetail, "CRITICAL: Fatal error") {
		t.Errorf("expected rclone detail in message: %q", withDetail)
	}
	if !strings.Contains(withDetail, "/tmp/RcloneManager.log") {
		t.Errorf("expected log path in message: %q", withDetail)
	}

	empty := mountFailureMessage(m, "", "/tmp/RcloneManager.log")
	if !strings.Contains(empty, "별도 오류 메시지를 출력하지 않았습니다") {
		t.Errorf("expected a fallback note when detail is empty: %q", empty)
	}
}
