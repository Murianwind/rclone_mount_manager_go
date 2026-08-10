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

func TestFetchLatestRelease_ParsesAssets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"tag_name": "v2.1.0",
			"assets": [
				{"name": "RcloneManager.zip", "browser_download_url": "https://example.com/RcloneManager.zip"}
			]
		}`))
	}))
	defer srv.Close()

	rel, err := FetchLatestRelease(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel.Version != "2.1.0" {
		t.Errorf("Version = %q, want %q", rel.Version, "2.1.0")
	}
	if len(rel.Assets) != 1 || rel.Assets[0].Name != "RcloneManager.zip" {
		t.Fatalf("unexpected assets: %+v", rel.Assets)
	}
	if rel.Assets[0].DownloadURL != "https://example.com/RcloneManager.zip" {
		t.Errorf("unexpected download URL: %q", rel.Assets[0].DownloadURL)
	}
}

func buildFakeAppZip(t *testing.T, exeContent []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("RcloneManager.exe")
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

func TestDownloadAppUpdate_NilClientDoesNotPanic(t *testing.T) {
	zipData := buildFakeAppZip(t, []byte("new-exe-bytes"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(zipData)
	}))
	defer srv.Close()

	// This exact call (client == nil) used to panic — client.Get() was
	// called on a nil *http.Client with no fallback, crashing the whole
	// app mid-update with nothing logged after "다운로드 시작".
	if _, err := DownloadAppUpdate(nil, t.TempDir(), srv.URL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDownloadAppUpdate_ExtractsExe(t *testing.T) {
	zipData := buildFakeAppZip(t, []byte("new-exe-bytes"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(zipData)
	}))
	defer srv.Close()

	destDir := t.TempDir()
	path, err := DownloadAppUpdate(srv.Client(), destDir, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != filepath.Join(destDir, "RcloneManager_new.exe") {
		t.Errorf("unexpected path: %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected extracted exe to exist: %v", err)
	}
	if string(data) != "new-exe-bytes" {
		t.Errorf("unexpected content: %q", data)
	}
}

func TestApplyUpdate_SwapsAndLaunches(t *testing.T) {
	dir := t.TempDir()
	currentExe := filepath.Join(dir, "RcloneManager.exe")
	newExe := filepath.Join(dir, "RcloneManager_new.exe")

	if err := os.WriteFile(currentExe, []byte("old-version"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newExe, []byte("new-version"), 0o755); err != nil {
		t.Fatal(err)
	}

	var launched string
	orig := launchFn
	launchFn = func(exePath string) error { launched = exePath; return nil }
	defer func() { launchFn = orig }()

	if err := ApplyUpdate(currentExe, newExe); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(currentExe)
	if err != nil {
		t.Fatalf("expected currentExe to exist after swap: %v", err)
	}
	if string(data) != "new-version" {
		t.Errorf("currentExe content = %q, want %q (the new version)", data, "new-version")
	}

	oldData, err := os.ReadFile(currentExe + ".old")
	if err != nil {
		t.Fatalf("expected the old exe to be preserved as .old: %v", err)
	}
	if string(oldData) != "old-version" {
		t.Errorf(".old content = %q, want %q", oldData, "old-version")
	}

	if launched != currentExe {
		t.Errorf("expected launch of %q, got %q", currentExe, launched)
	}
}

func TestApplyUpdate_RollsBackIfSwapFails(t *testing.T) {
	dir := t.TempDir()
	currentExe := filepath.Join(dir, "RcloneManager.exe")
	if err := os.WriteFile(currentExe, []byte("old-version"), 0o755); err != nil {
		t.Fatal(err)
	}
	missingNewExe := filepath.Join(dir, "does-not-exist.exe")

	orig := launchFn
	launchFn = func(exePath string) error { t.Fatalf("launchFn should not be called on failure"); return nil }
	defer func() { launchFn = orig }()

	if err := ApplyUpdate(currentExe, missingNewExe); err == nil {
		t.Fatalf("expected an error when the new exe doesn't exist")
	}

	data, err := os.ReadFile(currentExe)
	if err != nil {
		t.Fatalf("expected currentExe to be rolled back into place: %v", err)
	}
	if string(data) != "old-version" {
		t.Errorf("currentExe content after rollback = %q, want %q", data, "old-version")
	}
}
