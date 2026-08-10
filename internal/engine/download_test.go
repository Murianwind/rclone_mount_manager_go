package engine

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// buildFakeRcloneZip mirrors the shape of the real wiserain release asset:
// a zip containing "rclone-v<ver>-windows-amd64/rclone.exe".
func buildFakeRcloneZip(t *testing.T, exeContent []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("rclone-v1.65.0-windows-amd64/rclone.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(exeContent); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestDownloadRclone_InstallsDirectly(t *testing.T) {
	zipData := buildFakeRcloneZip(t, []byte("fake_exe"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(zipData)
	}))
	defer srv.Close()

	orig := rcloneDownloadURLFn
	rcloneDownloadURLFn = func(version string) string { return srv.URL }
	defer func() { rcloneDownloadURLFn = orig }()

	destDir := t.TempDir()
	status, err := DownloadRclone(srv.Client(), destDir, "1.65.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != DownloadStatusInstalled {
		t.Errorf("status = %q, want %q", status, DownloadStatusInstalled)
	}

	data, err := os.ReadFile(filepath.Join(destDir, "rclone.exe"))
	if err != nil {
		t.Fatalf("expected rclone.exe to be written: %v", err)
	}
	if string(data) != "fake_exe" {
		t.Errorf("rclone.exe content = %q, want %q", data, "fake_exe")
	}
}

func TestDownloadRclone_FallsBackToManualWhenTargetLocked(t *testing.T) {
	zipData := buildFakeRcloneZip(t, []byte("fake_exe_v2"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(zipData)
	}))
	defer srv.Close()

	orig := rcloneDownloadURLFn
	rcloneDownloadURLFn = func(version string) string { return srv.URL }
	defer func() { rcloneDownloadURLFn = orig }()

	destDir := t.TempDir()
	// Force the direct write to fail deterministically (on any OS/user,
	// including root) by making the target path a directory instead of a
	// regular file, simulating "can't replace rclone.exe right now".
	if err := os.Mkdir(filepath.Join(destDir, "rclone.exe"), 0o755); err != nil {
		t.Fatal(err)
	}

	status, err := DownloadRclone(srv.Client(), destDir, "1.65.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != DownloadStatusManual {
		t.Errorf("status = %q, want %q", status, DownloadStatusManual)
	}

	data, err := os.ReadFile(filepath.Join(destDir, "rclone_new.exe"))
	if err != nil {
		t.Fatalf("expected rclone_new.exe fallback to be written: %v", err)
	}
	if string(data) != "fake_exe_v2" {
		t.Errorf("rclone_new.exe content = %q, want %q", data, "fake_exe_v2")
	}
}

func TestDownloadRclone_MissingExeInZipIsAnError(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	_, _ = zw.Create("readme.txt")
	zw.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(buf.Bytes())
	}))
	defer srv.Close()

	orig := rcloneDownloadURLFn
	rcloneDownloadURLFn = func(version string) string { return srv.URL }
	defer func() { rcloneDownloadURLFn = orig }()

	if _, err := DownloadRclone(srv.Client(), t.TempDir(), "1.65.0"); err == nil {
		t.Errorf("expected an error when the zip has no rclone.exe")
	}
}

func TestDownloadAppRelease_SavesUnderFixedName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("fake update package"))
	}))
	defer srv.Close()

	destDir := t.TempDir()
	status, err := DownloadAppRelease(srv.Client(), destDir, srv.URL+"/RcloneManager-2.0.0.zip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != DownloadStatusManual {
		t.Errorf("status = %q, want %q", status, DownloadStatusManual)
	}

	data, err := os.ReadFile(filepath.Join(destDir, "RcloneManager_update.zip"))
	if err != nil {
		t.Fatalf("expected update file to be written: %v", err)
	}
	if string(data) != "fake update package" {
		t.Errorf("unexpected content: %q", data)
	}
}
