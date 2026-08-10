package engine

import "testing"

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

func count(list []string, item string) int {
	n := 0
	for _, v := range list {
		if v == item {
			n++
		}
	}
	return n
}

func TestBuildCmd_Basic(t *testing.T) {
	m := Mount{Remote: "drive", Drive: "X:", RemotePath: "data"}
	cmd := BuildCmd("rclone.exe", m)
	if !contains(cmd, "mount") {
		t.Errorf("expected cmd to contain \"mount\": %v", cmd)
	}
	if !contains(cmd, "drive:data") {
		t.Errorf("expected cmd to contain \"drive:data\": %v", cmd)
	}
}

func TestBuildCmd_WithCache(t *testing.T) {
	m := Mount{Remote: "drive", Drive: "X:", CacheDir: `C:\cache`, CacheMode: "full"}
	cmd := BuildCmd("rclone.exe", m)
	if !contains(cmd, "--cache-dir") || !contains(cmd, "full") {
		t.Errorf("expected cache flags in cmd: %v", cmd)
	}
}

func TestBuildCmd_WithExtraFlags(t *testing.T) {
	m := Mount{
		Remote:     "drive",
		Drive:      "X:",
		ExtraFlags: NormalizeFlags("--read-only; --bwlimit 10M"),
	}
	cmd := BuildCmd("rclone.exe", m)
	if !contains(cmd, "--read-only") || !contains(cmd, "--bwlimit=10M") {
		t.Errorf("expected normalized extra flags in cmd: %v", cmd)
	}
}

func TestBuildCmd_NoCacheFlagsWhenUnset(t *testing.T) {
	m := Mount{Remote: "drive", Drive: "X:"}
	cmd := BuildCmd("rclone.exe", m)
	if contains(cmd, "--cache-dir") || contains(cmd, "--vfs-cache-mode") {
		t.Errorf("did not expect cache flags: %v", cmd)
	}
}

func TestBuildCmd_VolnameDeduplication(t *testing.T) {
	m := Mount{
		Remote: "gds", Drive: "G:", RemotePath: "GDRIVE/VIDEO",
		ExtraFlags: "--volname=GDS;--no-modtime",
	}
	cmd := BuildCmd("rclone.exe", m)
	if got := count(cmd, "--volname"); got != 1 {
		t.Errorf("expected exactly one --volname flag, got %d in %v", got, cmd)
	}
	if !contains(cmd, "GDS") {
		t.Errorf("expected volname value GDS in %v", cmd)
	}
}

func TestGetVolname_FromExtraFlags(t *testing.T) {
	m := Mount{
		Remote: "gds", RemotePath: "GDRIVE/VIDEO",
		ExtraFlags: "--buffer-size=512M;--volname=GDS;--no-modtime",
	}
	if got := GetVolname(m); got != "GDS" {
		t.Errorf("GetVolname = %q, want GDS", got)
	}
}

func TestGetVolname_FromRemotePath(t *testing.T) {
	m := Mount{Remote: "PLEX", RemotePath: "KODI"}
	if got := GetVolname(m); got != "KODI" {
		t.Errorf("GetVolname = %q, want KODI", got)
	}
}

func TestGetVolname_FallbackToRemote(t *testing.T) {
	m := Mount{Remote: "nas"}
	if got := GetVolname(m); got != "nas" {
		t.Errorf("GetVolname = %q, want nas", got)
	}
}
