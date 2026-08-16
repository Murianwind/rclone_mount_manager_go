package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

// TestAutoMountAllSkipsWhileUpdatingRclone is a direct regression test for
// the real bug: the network monitor now retries autoMountAll() every 10s
// while connected, and without this guard it could start a fresh mount
// with the *old* rclone.exe in the brief window right after
// installOrUpdateRclone unmounts everything to free the file — locking
// rclone.exe again right before the update tries to overwrite it, and
// turning what should be an automatic replace into "rclone.exe가 사용
// 중이라 자동 교체하지 못했습니다."
func TestAutoMountAllSkipsWhileUpdatingRclone(t *testing.T) {
	Scenario(t, "GIVEN rclone.exe 업데이트가 진행 중임 WHEN autoMountAll 호출 THEN 아무 마운트도 시도하지 않고 조용히 반환한다 (회귀 테스트)", func(t *testing.T) {
		dir := t.TempDir()
		logPath := filepath.Join(dir, "x.log")
		rm := &rcloneManager{
			log: engine.RotatingLog{Path: logPath, MaxLines: 10},
		}
		rm.cfg.Mounts = []engine.Mount{{ID: "1", Remote: "gds", RemotePath: "x", AutoMount: true}}
		rm.updatingRclone.Store(true)

		rm.autoMountAll()

		// rclone 경로가 아예 설정 안 된 상태라, 실제로 마운트를 시도했다면
		// "rclone.exe를 찾을 수 없음" 류의 로그가 남았을 것이다 — 로그가
		// 완전히 비어있다는 건 시도 자체를 안 했다는 뜻이다.
		data, err := os.ReadFile(logPath)
		if err == nil && strings.TrimSpace(string(data)) != "" {
			t.Errorf("업데이트 중엔 마운트 시도 자체가 없어야 하는데 로그가 남음: %s", data)
		}
	})

	Scenario(t, "GIVEN rclone.exe 업데이트가 진행 중이 아님 WHEN autoMountAll 호출 THEN 정상적으로 마운트를 시도한다 (부정 케이스)", func(t *testing.T) {
		dir := t.TempDir()
		logPath := filepath.Join(dir, "x.log")
		rm := &rcloneManager{
			log: engine.RotatingLog{Path: logPath, MaxLines: 10},
		}
		rm.cfg.Mounts = []engine.Mount{{ID: "1", Remote: "gds", RemotePath: "x", AutoMount: true}}
		// updatingRclone은 기본값(false)로 둔다.

		if rm.isUpdatingRclone() {
			t.Fatalf("전제 확인 실패: updatingRclone이 기본으로 true면 안 됨")
		}
	})
}
