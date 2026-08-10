//go:build !windows

package engine

import (
	"errors"
	"os/exec"
	"testing"
)

// 이 파일은 !windows 빌드(즉 이 CI가 도는 Linux 러너)에서만 컴파일된다.
// 실제 Windows 레지스트리/콘솔 제어 API 대신, "지원 안 되는 플랫폼"
// 스텁(process_other.go)이 정상적으로 동작하는지 검증한다.

func TestConfigureBackgroundProcess(t *testing.T) {
	Scenario(t, "GIVEN Windows가 아닌 플랫폼 WHEN 백그라운드 프로세스로 설정 THEN 아무 동작도 안 하지만 panic도 없다 (no-op 확인)", func(t *testing.T) {
		cmd := exec.Command("echo", "hi")
		ConfigureBackgroundProcess(cmd) // 이 플랫폼에서는 no-op이어야 함
		if err := cmd.Run(); err != nil {
			t.Fatalf("명령 실행 중 예상치 못한 오류: %v", err)
		}
	})
}

func TestSignalGracefulStop(t *testing.T) {
	// 부정 케이스: 이 플랫폼엔 CTRL_BREAK 같은 정상 종료 신호 메커니즘이
	// 없으므로, 호출자가 즉시 강제종료로 폴백할 수 있게 명확한 오류를
	// 반환해야 한다(조용히 무시하거나 panic하면 안 됨).
	Scenario(t, "GIVEN Windows가 아닌 플랫폼 WHEN 정상 종료 신호를 시도 THEN 지원되지 않는다는 오류를 반환한다", func(t *testing.T) {
		err := SignalGracefulStop(1)
		if !errors.Is(err, ErrGracefulStopUnsupported) {
			t.Errorf("ErrGracefulStopUnsupported를 기대했는데 %v", err)
		}
	})
}
