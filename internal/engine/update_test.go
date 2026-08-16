package engine

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── 키워드 ──

// buildFakeAppZip은 앱 자체 업데이트 자산과 같은 구조("RcloneManager.exe"를
// 담은 zip)를 만든다.
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

func givenOldExeAt(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func withFakeLaunch(t *testing.T) *string {
	t.Helper()
	var launched string
	orig := launchFn
	launchFn = func(exePath string) error { launched = exePath; return nil }
	t.Cleanup(func() { launchFn = orig })
	return &launched
}

func TestDownloadAppUpdate(t *testing.T) {
	// 회귀 테스트: client == nil로 호출하면 client.Get()이 nil 포인터
	// 역참조로 panic → 복구되지 않은 고루틴 패닉 → 앱 전체가 조용히
	// 죽는 실제 버그가 있었다("다운로드 시작" 로그 이후 아무 것도 안
	// 남던 원인). 이 부정 케이스가 다시 발생하지 않는지 고정해둔다.
	Scenario(t, "GIVEN client가 nil로 전달됨(과거 실제 크래시 버그) WHEN 업데이트 다운로드 THEN panic 없이 기본 클라이언트로 정상 동작한다 (부정 케이스/회귀 테스트)", func(t *testing.T) {
		zipData := buildFakeAppZip(t, []byte("new-exe-bytes"))
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(zipData)
		}))
		defer srv.Close()

		if _, err := DownloadAppUpdate(nil, t.TempDir(), srv.URL); err != nil {
			t.Fatalf("예상치 못한 오류: %v", err)
		}
	})

	Scenario(t, "GIVEN RcloneManager.exe가 든 정상 업데이트 zip WHEN 다운로드 THEN RcloneManager_new.exe로 추출된다", func(t *testing.T) {
		zipData := buildFakeAppZip(t, []byte("new-exe-bytes"))
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(zipData)
		}))
		defer srv.Close()

		destDir := t.TempDir()
		path, err := DownloadAppUpdate(srv.Client(), destDir, srv.URL)
		thenNoError(t, err)
		if path != filepath.Join(destDir, "RcloneManager_new.exe") {
			t.Errorf("예상치 못한 경로: %q", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("추출된 exe 파일이 있어야 함: %v", err)
		}
		if string(data) != "new-exe-bytes" {
			t.Errorf("예상치 못한 내용: %q", data)
		}
	})
}

func TestCleanupPreviousExe(t *testing.T) {
	Scenario(t, "GIVEN 이전 업데이트가 남긴 .old 백업 파일이 있음 WHEN 정리 실행 THEN 그 파일을 삭제한다", func(t *testing.T) {
		dir := t.TempDir()
		currentExe := filepath.Join(dir, "RcloneManager.exe")
		oldFile := currentExe + ".old"
		givenOldExeAt(t, oldFile, "leftover")

		CleanupPreviousExe(currentExe)

		if _, err := os.Stat(oldFile); err == nil {
			t.Errorf(".old 파일이 삭제됐어야 함")
		}
	})

	// 부정 케이스: 정리할 .old 파일이 애초에 없는(정상적인 최초 실행) 상황.
	Scenario(t, "GIVEN .old 파일이 애초에 없음 WHEN 정리 실행 THEN panic 없이 조용히 넘어간다 (부정 케이스)", func(t *testing.T) {
		dir := t.TempDir()
		currentExe := filepath.Join(dir, "RcloneManager.exe")
		CleanupPreviousExe(currentExe) // 존재하지 않아도 오류 없이 그냥 지나가야 함
	})

	// 부정 케이스: .old 자리가 절대 지워지지 않는 상태로 계속 유지되는
	// 상황(실제로는 다른 프로세스가 계속 잠그고 있는 경우에 대응) —
	// 무한 재시도로 멈춰버리면 안 되고, 정해진 횟수만 시도하다 포기해야
	// 한다. 이 테스트 환경에서는 진짜 파일 잠금을 재현할 수 없어서,
	// os.Remove가 항상 실패하는 상황(내용이 있는 디렉터리)으로 대신한다.
	Scenario(t, "GIVEN .old 자리가 계속 삭제 불가능한 상태임 WHEN 정리 실행 THEN 무한정 재시도하지 않고 정해진 횟수 안에 포기한다 (부정 케이스)", func(t *testing.T) {
		dir := t.TempDir()
		currentExe := filepath.Join(dir, "RcloneManager.exe")
		oldPath := currentExe + ".old"
		if err := os.Mkdir(oldPath, 0o755); err != nil {
			t.Fatal(err)
		}
		// 빈 디렉터리면 Remove가 성공해버리니, 내용물을 하나 넣어 항상
		// 실패하게 만든다.
		if err := os.WriteFile(filepath.Join(oldPath, "x"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}

		start := time.Now()
		CleanupPreviousExe(currentExe)
		elapsed := time.Since(start)

		if elapsed > 5*time.Second {
			t.Errorf("재시도가 너무 오래 걸림(%v) — 정해진 횟수 안에 포기해야 함", elapsed)
		}
		if _, err := os.Stat(oldPath); err != nil {
			t.Errorf(".old 자리가 실제로 안 지워진 상황을 재현하려던 건데 이미 없어짐: %v", err)
		}
	})
}

func TestApplyUpdate(t *testing.T) {
	Scenario(t, "GIVEN 실행 중인 구버전 exe와 다운로드된 신버전 exe WHEN 업데이트 적용 THEN 구버전은 .old로 보존되고 신버전이 그 자리를 대체하며 재실행된다", func(t *testing.T) {
		dir := t.TempDir()
		currentExe := filepath.Join(dir, "RcloneManager.exe")
		newExe := filepath.Join(dir, "RcloneManager_new.exe")
		givenOldExeAt(t, currentExe, "old-version")
		givenOldExeAt(t, newExe, "new-version")
		launched := withFakeLaunch(t)

		err := ApplyUpdate(currentExe, newExe)
		thenNoError(t, err)

		data, err := os.ReadFile(currentExe)
		if err != nil {
			t.Fatalf("교체 후 currentExe가 있어야 함: %v", err)
		}
		if string(data) != "new-version" {
			t.Errorf("currentExe 내용 = %q, 기대값(신버전) %q", data, "new-version")
		}

		oldData, err := os.ReadFile(currentExe + ".old")
		if err != nil {
			t.Fatalf("구버전이 .old로 보존됐어야 함: %v", err)
		}
		if string(oldData) != "old-version" {
			t.Errorf(".old 내용 = %q, 기대값 %q", oldData, "old-version")
		}

		if *launched != currentExe {
			t.Errorf("%q가 재실행됐어야 하는데 %q", currentExe, *launched)
		}
	})

	// 부정 케이스: 신버전 파일이 없어서(다운로드 실패 등) 교체 자체가
	// 안 되는 상황 — 이때 기존 exe를 잃어버리면 안 되고 롤백돼야 한다.
	Scenario(t, "GIVEN 신버전 exe 파일이 존재하지 않음 WHEN 업데이트 적용 시도 THEN 오류를 반환하고 기존 exe를 원래 자리로 롤백한다 (부정 케이스)", func(t *testing.T) {
		dir := t.TempDir()
		currentExe := filepath.Join(dir, "RcloneManager.exe")
		givenOldExeAt(t, currentExe, "old-version")
		missingNewExe := filepath.Join(dir, "does-not-exist.exe")

		orig := launchFn
		launchFn = func(exePath string) error { t.Fatalf("실패 시에는 launchFn이 호출되면 안 됨"); return nil }
		defer func() { launchFn = orig }()

		if err := ApplyUpdate(currentExe, missingNewExe); err == nil {
			t.Fatalf("신버전 exe가 없으면 오류가 나야 함")
		}

		data, err := os.ReadFile(currentExe)
		if err != nil {
			t.Fatalf("currentExe가 원래 자리로 롤백돼 있어야 함: %v", err)
		}
		if string(data) != "old-version" {
			t.Errorf("롤백 후 currentExe 내용 = %q, 기대값 %q", data, "old-version")
		}
	})
}
