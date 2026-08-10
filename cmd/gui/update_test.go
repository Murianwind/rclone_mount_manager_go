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

	if got := findAsset(assets, "RcloneManager.zip"); got != "https://example.com/RcloneManager.zip" {
		t.Errorf("findAsset = %q, want the RcloneManager.zip URL", got)
	}
	if got := findAsset(assets, "does-not-exist.zip"); got != "" {
		t.Errorf("findAsset for a missing name = %q, want \"\"", got)
	}
	if got := findAsset(nil, "RcloneManager.zip"); got != "" {
		t.Errorf("findAsset on nil assets = %q, want \"\"", got)
	}
}
