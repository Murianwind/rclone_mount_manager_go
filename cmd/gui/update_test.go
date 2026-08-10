package main

import (
	"testing"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func TestFindAsset(t *testing.T) {
	assets := []engine.ReleaseAsset{
		{Name: "checksums.txt", DownloadURL: "https://example.com/checksums.txt"},
		{Name: "RcloneManager.zip", DownloadURL: "https://example.com/RcloneManager.zip"},
	}

	Scenario(t, "GIVEN 릴리스 자산 목록에 원하는 이름이 있음 WHEN 자산 검색 THEN 해당 다운로드 URL을 반환한다", func(t *testing.T) {
		got := findAsset(assets, "RcloneManager.zip")
		if got != "https://example.com/RcloneManager.zip" {
			t.Errorf("got %q, RcloneManager.zip의 URL이어야 함", got)
		}
	})

	// 부정 케이스: 릴리스가 발행됐지만 원하는 자산 이름이 없는 상황
	// (예: 빌드 워크플로가 다른 이름으로 올린 경우).
	Scenario(t, "GIVEN 원하는 이름의 자산이 목록에 없음 WHEN 자산 검색 THEN 빈 문자열을 반환한다 (부정 케이스)", func(t *testing.T) {
		got := findAsset(assets, "does-not-exist.zip")
		if got != "" {
			t.Errorf("got %q, 찾는 자산이 없으면 빈 문자열이어야 함", got)
		}
	})

	Scenario(t, "GIVEN 자산 목록 자체가 nil임(릴리스에 자산이 아예 없음) WHEN 자산 검색 THEN 빈 문자열을 반환한다 (부정 케이스)", func(t *testing.T) {
		got := findAsset(nil, "RcloneManager.zip")
		if got != "" {
			t.Errorf("got %q, 자산이 없으면 빈 문자열이어야 함", got)
		}
	})
}
