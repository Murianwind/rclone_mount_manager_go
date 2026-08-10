package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ── 키워드 ──

// givenStoreAt는 dir을 대상 디렉토리로 하는 Store를 만든다.
func givenStoreAt(dir string) Store { return Store{Dir: dir} }

// givenExistingFile은 테스트 시작 전 미리 파일을 하나 만들어둔다
// (손상된 mounts.json, 백업 파일 등 사전 상태를 준비할 때 사용).
func givenExistingFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func whenLoaded(s Store) (Config, error) { return s.Load() }

func whenSaved(s Store, cfg Config) error { return s.Save(cfg) }

func thenNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("예상치 못한 오류: %v", err)
	}
}

func thenMountCount(t *testing.T, cfg Config, want int) {
	t.Helper()
	if len(cfg.Mounts) != want {
		t.Errorf("마운트 개수 = %d, 기대값 %d (%+v)", len(cfg.Mounts), want, cfg.Mounts)
	}
}

func TestLoad(t *testing.T) {
	Scenario(t, "GIVEN mounts.json 파일이 아예 없음 WHEN 로드 THEN 빈 기본값을 반환한다", func(t *testing.T) {
		s := givenStoreAt(t.TempDir())
		cfg, err := whenLoaded(s)
		thenNoError(t, err)
		thenMountCount(t, cfg, 0)
	})

	Scenario(t, "GIVEN mounts.json이 손상됐고 백업도 없음 WHEN 로드 THEN 기본값을 반환하고 손상 파일은 삭제하지 않고 보존한다 (부정 케이스)", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "mounts.json")
		givenExistingFile(t, path, "{bad")

		cfg, err := whenLoaded(givenStoreAt(dir))
		thenNoError(t, err)
		thenMountCount(t, cfg, 0)

		// 원본 mounts.json은 사라져야 하지만(rename), 조용히 삭제되면 안 된다.
		if _, err := os.Stat(path); err == nil {
			t.Errorf("손상된 mounts.json이 다른 이름으로 옮겨졌어야 하는데 그대로 남아있음")
		}
		matches, _ := filepath.Glob(filepath.Join(dir, "mounts.json.corrupted-*"))
		if len(matches) != 1 {
			t.Errorf("보존된 손상 파일이 정확히 1개여야 하는데 %v", matches)
		}
	})

	Scenario(t, "GIVEN mounts.json은 손상됐지만 mounts.json.bak은 정상 WHEN 로드 THEN 백업에서 자동 복구한다", func(t *testing.T) {
		dir := t.TempDir()
		givenExistingFile(t, filepath.Join(dir, "mounts.json"), "{bad")
		givenExistingFile(t, filepath.Join(dir, "mounts.json.bak"), `{"mounts":[{"remote":"backup-ok"}]}`)

		cfg, err := whenLoaded(givenStoreAt(dir))
		thenNoError(t, err)
		thenMountCount(t, cfg, 1)
		if cfg.Mounts[0].Remote != "backup-ok" {
			t.Errorf("백업에서 복구된 리모트 = %q, 기대값 %q", cfg.Mounts[0].Remote, "backup-ok")
		}
	})

	// 부정 케이스: 원본도 백업도 둘 다 손상된, 복구 불가능한 최악의 상황.
	Scenario(t, "GIVEN mounts.json과 백업 둘 다 손상됨 WHEN 로드 THEN 원본을 보존한 채 기본값을 반환한다 (복구 불가 시나리오)", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "mounts.json")
		givenExistingFile(t, path, "{bad")
		givenExistingFile(t, filepath.Join(dir, "mounts.json.bak"), "{also bad")

		cfg, err := whenLoaded(givenStoreAt(dir))
		thenNoError(t, err)
		thenMountCount(t, cfg, 0)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("손상된 원본이 그대로 남아있으면 안 됨 (다른 이름으로 옮겨졌어야 함)")
		}
	})

	Scenario(t, "GIVEN 정상적인 mounts.json (mounts 키 존재) WHEN 로드 THEN 있는 그대로 파싱된다", func(t *testing.T) {
		dir := t.TempDir()
		givenExistingFile(t, filepath.Join(dir, "mounts.json"), `{"mounts":[{"remote":"x"}],"remotes":[]}`)

		cfg, err := whenLoaded(givenStoreAt(dir))
		thenNoError(t, err)
		thenMountCount(t, cfg, 1)
	})
}

func TestSave(t *testing.T) {
	Scenario(t, "GIVEN 기존 파일이 없음 WHEN 저장 THEN 원자적으로(임시파일 없이) mounts.json이 생성된다", func(t *testing.T) {
		dir := t.TempDir()
		err := whenSaved(givenStoreAt(dir), Config{Mounts: []Mount{}})
		thenNoError(t, err)

		path := filepath.Join(dir, "mounts.json")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("mounts.json이 생성됐어야 함: %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, "mounts.json.tmp")); err == nil {
			t.Errorf("원자적 rename 후에는 임시 파일이 남아있으면 안 됨")
		}
	})

	Scenario(t, "GIVEN 기존에 유효한 mounts.json이 있음 WHEN 새 설정을 저장 THEN 기존 내용이 .bak으로 백업되고 새 내용으로 교체된다", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "mounts.json")
		givenExistingFile(t, path, `{"mounts":[{"remote":"old"}]}`)

		err := whenSaved(givenStoreAt(dir), Config{Mounts: []Mount{{Remote: "new"}}})
		thenNoError(t, err)

		backup, err := os.ReadFile(filepath.Join(dir, "mounts.json.bak"))
		if err != nil {
			t.Fatalf("백업 파일이 있어야 함: %v", err)
		}
		var backupCfg Config
		if err := json.Unmarshal(backup, &backupCfg); err != nil {
			t.Fatalf("백업 파일이 유효한 JSON이 아님: %v", err)
		}
		if len(backupCfg.Mounts) != 1 || backupCfg.Mounts[0].Remote != "old" {
			t.Errorf("백업에는 옛 설정(old)이 들어있어야 하는데 %+v", backupCfg)
		}

		current, _ := os.ReadFile(path)
		var currentCfg Config
		_ = json.Unmarshal(current, &currentCfg)
		if len(currentCfg.Mounts) != 1 || currentCfg.Mounts[0].Remote != "new" {
			t.Errorf("mounts.json에는 새 설정(new)이 들어있어야 하는데 %+v", currentCfg)
		}
	})
}
