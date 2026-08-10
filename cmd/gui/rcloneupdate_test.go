package main

import (
	"testing"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func TestActiveMountsSnapshot(t *testing.T) {
	Scenario(t, "GIVEN 마운트 3개 중 1개만 현재 실행 중임 WHEN 실행 중 마운트 스냅샷 조회 THEN 실행 중인 것만 반환한다", func(t *testing.T) {
		rm := &rcloneManager{
			active: map[string]*runningMount{"m2": {}},
		}
		rm.cfg.Mounts = []engine.Mount{{ID: "m1"}, {ID: "m2"}, {ID: "m3"}}

		got := rm.activeMountsSnapshot()
		if len(got) != 1 || got[0].ID != "m2" {
			t.Errorf("got %+v, 기대값 [m2]만 포함", got)
		}
	})

	// 부정 케이스: 실행 중인 마운트가 하나도 없는 상태.
	Scenario(t, "GIVEN 실행 중인 마운트가 하나도 없음 WHEN 스냅샷 조회 THEN 빈 목록을 반환한다 (부정 케이스)", func(t *testing.T) {
		rm := &rcloneManager{active: map[string]*runningMount{}}
		rm.cfg.Mounts = []engine.Mount{{ID: "m1"}}

		if got := rm.activeMountsSnapshot(); len(got) != 0 {
			t.Errorf("빈 목록을 기대했는데 %+v", got)
		}
	})
}
