package engine

import (
	"os"
	"path/filepath"
	"testing"
)

// ── 키워드 ──

func givenConfFile(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "rclone.conf")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func whenConfParsed(path string) []Remote { return ParseRcloneConf(path) }

func thenRemoteAt(t *testing.T, remotes []Remote, i int, wantName, wantType string) {
	t.Helper()
	if i >= len(remotes) {
		t.Fatalf("remotes[%d]가 없음 (총 %d개): %+v", i, len(remotes), remotes)
	}
	if remotes[i].Name != wantName || remotes[i].Type != wantType {
		t.Errorf("remotes[%d] = %+v, 기대값 {%q %q}", i, remotes[i], wantName, wantType)
	}
}

func TestParseRcloneConf(t *testing.T) {
	Scenario(t, "GIVEN 여러 섹션이 있는 정상적인 rclone.conf WHEN 파싱 THEN 각 리모트의 이름·타입을 순서대로 추출한다", func(t *testing.T) {
		path := givenConfFile(t, t.TempDir(), `[PLEX]
type = drive
client_id = xxx

[nas]
type = sftp
host = 192.168.0.10
`)
		remotes := whenConfParsed(path)
		if len(remotes) != 2 {
			t.Fatalf("리모트 개수 = %d, 기대값 2 (%+v)", len(remotes), remotes)
		}
		thenRemoteAt(t, remotes, 0, "PLEX", "drive")
		thenRemoteAt(t, remotes, 1, "nas", "sftp")
	})

	// 부정 케이스: 파일 자체가 없는 상황(아직 rclone을 설정 안 한 사용자 등).
	Scenario(t, "GIVEN 파일이 존재하지 않음 WHEN 파싱 THEN panic이나 오류 없이 빈 목록을 반환한다 (부정 케이스)", func(t *testing.T) {
		remotes := whenConfParsed(filepath.Join(t.TempDir(), "no-such-file.conf"))
		if len(remotes) != 0 {
			t.Errorf("빈 목록을 기대했는데 %+v", remotes)
		}
	})

	Scenario(t, "GIVEN type 줄이 없는 섹션(설정이 불완전한 리모트) WHEN 파싱 THEN 타입은 빈 문자열로 남기고 이름은 정상 추출한다 (경계 케이스)", func(t *testing.T) {
		path := givenConfFile(t, t.TempDir(), "[broken]\nsome_other_key = value\n")
		remotes := whenConfParsed(path)
		thenRemoteAt(t, remotes, 0, "broken", "")
	})

	Scenario(t, "GIVEN 섹션 헤더 앞에 나오는 줄(빈 파일 맨 앞 주석 등) WHEN 파싱 THEN 무시하고 panic도 없다 (경계 케이스)", func(t *testing.T) {
		path := givenConfFile(t, t.TempDir(), "; 이것은 주석입니다\n\n[first]\ntype = drive\n")
		remotes := whenConfParsed(path)
		thenRemoteAt(t, remotes, 0, "first", "drive")
	})
}

func TestFindDefaultRcloneConf(t *testing.T) {
	Scenario(t, "GIVEN appDir 폴더에 rclone.conf가 있고 다른 후보 경로엔 없음 WHEN 기본 경로 탐색 THEN appDir의 파일을 최후 폴백으로 찾아낸다", func(t *testing.T) {
		appDir := t.TempDir()
		// APPDATA/홈 디렉터리에 실제로 rclone.conf가 있을 가능성을 배제할
		// 수 없는 CI 환경이라도, appDir 후보가 항상 우선순위 마지막이므로
		// 이 시나리오는 "적어도 appDir 폴백은 동작한다"만 확인한다.
		givenConfFile(t, appDir, "[x]\ntype = drive\n")

		// APPDATA를 존재하지 않는 경로로 강제해 그 후보를 확실히 배제한다.
		t.Setenv("APPDATA", filepath.Join(appDir, "no-such-appdata"))

		got, ok := FindDefaultRcloneConf(appDir)
		if !ok {
			t.Fatalf("appDir 폴백을 찾았어야 함")
		}
		if got != filepath.Join(appDir, "rclone.conf") {
			t.Errorf("got %q, 기대값 appDir 안의 rclone.conf", got)
		}
	})

	// 부정 케이스: 어느 후보 경로에도 파일이 없는 상황.
	Scenario(t, "GIVEN 어떤 후보 경로에도 rclone.conf가 없음 WHEN 기본 경로 탐색 THEN 찾지 못했음을 반환한다 (부정 케이스)", func(t *testing.T) {
		appDir := t.TempDir()
		t.Setenv("APPDATA", filepath.Join(appDir, "no-such-appdata"))
		t.Setenv("HOME", filepath.Join(appDir, "no-such-home"))
		t.Setenv("USERPROFILE", filepath.Join(appDir, "no-such-home")) // Windows의 os.UserHomeDir()가 참조

		if _, ok := FindDefaultRcloneConf(filepath.Join(appDir, "empty-app-dir")); ok {
			t.Errorf("아무 후보에도 파일이 없으면 ok=false여야 함")
		}
	})
}
