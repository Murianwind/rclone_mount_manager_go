package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── 키워드 ──

func givenLogFileWithLines(t *testing.T, path string, lines ...string) {
	t.Helper()
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func whenLogWritten(l RotatingLog, level, message string) error {
	return l.Write(level, message)
}

func thenLogHasLineCount(t *testing.T, path string, want int) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("예상치 못한 오류: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != want {
		t.Fatalf("rotate 후 줄 수 = %d, 기대값 %d (%v)", len(lines), want, lines)
	}
	return lines
}

func TestRotatingLog(t *testing.T) {
	Scenario(t, "GIVEN 최대 줄 수를 초과한 로그 파일 WHEN 한 줄을 추가로 기록 THEN 가장 오래된 줄이 삭제되고 최대 줄 수를 유지한다", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "RcloneManager.log")
		givenLogFileWithLines(t, path, "l1", "l2", "l3")

		l := RotatingLog{Path: path, MaxLines: 3}
		err := whenLogWritten(l, "INFO", "새 라인")
		thenNoError(t, err)

		lines := thenLogHasLineCount(t, path, 3)
		if lines[0] != "l2" {
			t.Errorf("가장 오래 살아남은 줄 = %q, 기대값 %q (l1이 삭제됐어야 함)", lines[0], "l2")
		}
		if !strings.Contains(lines[2], "새 라인") {
			t.Errorf("가장 최근 줄에 새 메시지가 포함돼야 하는데 %q", lines[2])
		}
	})

	// 부정 케이스: 로그 기록 자체가 실패하는 상황(예: 디스크 문제로 경로가
	// 파일이 아니게 됨)에서도 앱이 죽으면 안 된다는 게 write_log의 핵심 의도.
	Scenario(t, "GIVEN 로그 경로에 쓸 수 없는 상황(디렉터리를 파일 경로로 지정) WHEN 로그 기록 시도 THEN panic 없이 오류만 반환한다 (부정 케이스)", func(t *testing.T) {
		dir := t.TempDir()
		l := RotatingLog{Path: dir, MaxLines: 10} // 디렉터리를 파일처럼 쓰려는 잘못된 설정

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Write가 panic을 일으킴: %v", r)
				}
			}()
			if err := whenLogWritten(l, "ERROR", "실패해도 괜찮음"); err == nil {
				t.Errorf("디렉터리 경로에 쓰면 오류가 나야 하는데 nil이 반환됨")
			}
		}()
	})
}
