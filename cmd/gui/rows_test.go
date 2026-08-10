package main

import (
	"testing"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func TestKindLabel(t *testing.T) {
	Scenario(t, "GIVEN 마운트 행 WHEN 구분 라벨 결정 THEN '💾 마운트'를 보여준다", func(t *testing.T) {
		if got := kindLabel(rowKindMount); got != "💾 마운트" {
			t.Errorf("got %q, 기대값 %q", got, "💾 마운트")
		}
	})
	Scenario(t, "GIVEN 원본 리모트 행 WHEN 구분 라벨 결정 THEN '☁️ 원본'을 보여준다", func(t *testing.T) {
		if got := kindLabel(rowKindRemote); got != "☁️ 원본" {
			t.Errorf("got %q, 기대값 %q", got, "☁️ 원본")
		}
	})
}

func TestRemoteDisplayText(t *testing.T) {
	Scenario(t, "GIVEN 이름과 타입이 있는 원본 리모트 WHEN 표시 텍스트 구성 THEN '[타입] 이름' 형식이 된다", func(t *testing.T) {
		got := remoteDisplayText(engine.Remote{Name: "PLEX", Type: "drive"})
		if got != "[drive] PLEX" {
			t.Errorf("got %q, 기대값 %q", got, "[drive] PLEX")
		}
	})
}

func TestRcloneManagerRows(t *testing.T) {
	Scenario(t, "GIVEN 원본 리모트 2개와 마운트 1개가 설정된 상태 WHEN 통합 행 목록 조회 THEN 원본이 먼저, 마운트가 그 뒤에 온다", func(t *testing.T) {
		rm := &rcloneManager{}
		rm.cfg.Remotes = []engine.Remote{{Name: "a"}, {Name: "b"}}
		rm.cfg.Mounts = []engine.Mount{{ID: "m1"}}

		rows := rm.rows()
		if len(rows) != 3 {
			t.Fatalf("행 개수 = %d, 기대값 3", len(rows))
		}
		if rows[0].kind != rowKindRemote || rows[1].kind != rowKindRemote {
			t.Errorf("앞 두 행은 원본이어야 함: %+v", rows[:2])
		}
		if rows[2].kind != rowKindMount || rows[2].mount.ID != "m1" {
			t.Errorf("마지막 행은 마운트여야 함: %+v", rows[2])
		}
	})

	// 부정/경계 케이스: 원본도 마운트도 하나도 없는 초기 상태.
	Scenario(t, "GIVEN 원본도 마운트도 하나도 없음 WHEN 통합 행 목록 조회 THEN 빈 목록을 반환한다 (경계 케이스)", func(t *testing.T) {
		rm := &rcloneManager{}
		if got := rm.rows(); len(got) != 0 {
			t.Errorf("빈 목록을 기대했는데 %+v", got)
		}
	})
}
