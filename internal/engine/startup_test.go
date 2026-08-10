package engine

import (
	"errors"
	"strings"
	"testing"
)

// 이 CI의 Linux 러너에서는 빌드 태그로 선택되는 구현이 !windows
// 스텁(startup_other.go)이다 — Python 테스트가 rclone_manager.winreg를
// None으로 패치해서 검증하던 것과 같은 역할이다. 실제 Windows 빌드에서는
// startup_windows.go가 대신 컴파일되어 진짜 레지스트리 경로를 탄다.

// ── 키워드 ──

func whenCheckingStartupEnabled() bool { return IsStartupEnabled() }

func whenReadingStartupPath() string { return GetStartupPath() }

func whenSettingStartup(enable bool) error { return SetStartup(enable) }

// withStartupFns는 CheckAndFixStartup이 내부적으로 호출하는 함수들을
// 테스트용으로 바꿔치기한다 — 실제 레지스트리 없이도 분기 로직을
// 검증하기 위한 키워드 주도 테스트의 "테스트 대역(더블)" 역할.
func withStartupFns(t *testing.T, pathFn func() string, setFn func(bool) error, exeFn func() (string, error)) {
	t.Helper()
	origPath, origSet, origExe := startupPathFn, setStartupFn, currentExePathFn
	startupPathFn, setStartupFn, currentExePathFn = pathFn, setFn, exeFn
	t.Cleanup(func() {
		startupPathFn, setStartupFn, currentExePathFn = origPath, origSet, origExe
	})
}

func TestStartupRegistry_NoRegistryOnThisPlatform(t *testing.T) {
	// 부정 케이스 3종: 이 플랫폼엔 진짜 레지스트리가 없으므로 셋 다
	// "안전하게 실패"해야 한다 — panic도, 거짓 성공도 없어야 함.
	Scenario(t, "GIVEN 이 플랫폼엔 Windows 레지스트리가 없음 WHEN 등록 여부 확인 THEN false를 반환한다", func(t *testing.T) {
		if whenCheckingStartupEnabled() {
			t.Errorf("실제 Windows 레지스트리가 없으니 false여야 함")
		}
	})

	Scenario(t, "GIVEN 이 플랫폼엔 Windows 레지스트리가 없음 WHEN 등록된 경로 조회 THEN 빈 문자열을 반환한다", func(t *testing.T) {
		if got := whenReadingStartupPath(); got != "" {
			t.Errorf("GetStartupPath() = %q, 기대값 \"\"", got)
		}
	})

	Scenario(t, "GIVEN 이 플랫폼엔 Windows 레지스트리가 없음 WHEN 시작프로그램 등록 시도 THEN 오류를 반환한다", func(t *testing.T) {
		if err := whenSettingStartup(true); err == nil {
			t.Errorf("실제 레지스트리가 없으니 오류가 나야 함")
		}
	})
}

func TestGetCurrentExePath(t *testing.T) {
	Scenario(t, "GIVEN 현재 실행 파일 경로 WHEN 조회 THEN 레지스트리 Run 키 형식에 맞게 큰따옴표로 감싼 값을 반환한다", func(t *testing.T) {
		path, err := GetCurrentExePath()
		thenNoError(t, err)
		if !strings.HasPrefix(path, `"`) || !strings.HasSuffix(path, `"`) {
			t.Errorf("따옴표로 감싼 경로를 기대했는데 %q", path)
		}
	})
}

func TestCheckAndFixStartup(t *testing.T) {
	Scenario(t, "GIVEN 시작프로그램에 등록된 적이 없음 WHEN 확인·수정 시도 THEN 아무것도 하지 않는다", func(t *testing.T) {
		withStartupFns(t,
			func() string { return "" },
			func(bool) error { return nil },
			func() (string, error) { return "current", nil },
		)
		changed, err := CheckAndFixStartup()
		thenNoError(t, err)
		if changed {
			t.Errorf("등록된 적이 없으면 변경이 없어야 함")
		}
	})

	Scenario(t, "GIVEN 등록된 경로가 현재 실행 파일 경로와 일치함 WHEN 확인·수정 시도 THEN 재등록하지 않는다", func(t *testing.T) {
		withStartupFns(t,
			func() string { return "samepath" },
			func(bool) error { t.Fatalf("경로가 일치하면 SetStartup이 호출되면 안 됨"); return nil },
			func() (string, error) { return "samepath", nil },
		)
		changed, err := CheckAndFixStartup()
		thenNoError(t, err)
		if changed {
			t.Errorf("경로가 이미 일치하면 변경이 없어야 함")
		}
	})

	Scenario(t, "GIVEN 등록된 경로가 현재 실행 파일 경로와 다름(exe 이동/업데이트됨) WHEN 확인·수정 시도 THEN 새 경로로 재등록한다", func(t *testing.T) {
		var setCalledWith *bool
		withStartupFns(t,
			func() string { return "old" },
			func(enable bool) error { setCalledWith = &enable; return nil },
			func() (string, error) { return "new", nil },
		)
		changed, err := CheckAndFixStartup()
		thenNoError(t, err)
		if !changed {
			t.Errorf("경로가 다르면 재등록(변경)이 일어나야 함")
		}
		if setCalledWith == nil || *setCalledWith != true {
			t.Errorf("SetStartup(true)로 호출됐어야 함")
		}
	})

	// 부정 케이스: 레지스트리 쓰기 자체가 실패하는 상황(권한 문제 등).
	Scenario(t, "GIVEN 재등록이 필요한데 SetStartup 자체가 실패함 WHEN 확인·수정 시도 THEN 그 오류가 그대로 호출자에게 전달된다 (부정 케이스)", func(t *testing.T) {
		wantErr := errors.New("boom")
		withStartupFns(t,
			func() string { return "old" },
			func(bool) error { return wantErr },
			func() (string, error) { return "new", nil },
		)
		_, err := CheckAndFixStartup()
		if !errors.Is(err, wantErr) {
			t.Errorf("내부 오류가 그대로 전파돼야 하는데 %v", err)
		}
	})
}
