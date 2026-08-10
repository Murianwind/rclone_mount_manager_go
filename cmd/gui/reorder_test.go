package main

import (
	"testing"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func TestSwapMountsUpDown(t *testing.T) {
	Scenario(t, "GIVEN 마운트 목록의 중간 항목 WHEN 위로 이동 THEN 바로 앞 항목과 자리가 바뀐다", func(t *testing.T) {
		mounts := []engine.Mount{{ID: "a"}, {ID: "b"}, {ID: "c"}}
		ok := swapMountsUp(mounts, 1)
		if !ok {
			t.Fatalf("이동이 성공했어야 함")
		}
		if mounts[0].ID != "b" || mounts[1].ID != "a" {
			t.Errorf("순서 = %v, 기대값 [b a c]", mountIDs(mounts))
		}
	})

	// 부정 케이스: 이미 맨 위에 있는 항목은 더 위로 이동할 수 없다.
	Scenario(t, "GIVEN 이미 맨 위에 있는 항목(인덱스 0) WHEN 위로 이동 시도 THEN 아무 변화 없이 실패를 반환한다 (경계 케이스)", func(t *testing.T) {
		mounts := []engine.Mount{{ID: "a"}, {ID: "b"}}
		ok := swapMountsUp(mounts, 0)
		if ok {
			t.Errorf("맨 위 항목은 위로 이동할 수 없어야 함")
		}
		if mounts[0].ID != "a" || mounts[1].ID != "b" {
			t.Errorf("순서가 바뀌면 안 되는데 %v", mountIDs(mounts))
		}
	})

	Scenario(t, "GIVEN 마운트 목록의 중간 항목 WHEN 아래로 이동 THEN 바로 뒤 항목과 자리가 바뀐다", func(t *testing.T) {
		mounts := []engine.Mount{{ID: "a"}, {ID: "b"}, {ID: "c"}}
		ok := swapMountsDown(mounts, 1)
		if !ok {
			t.Fatalf("이동이 성공했어야 함")
		}
		if mounts[1].ID != "c" || mounts[2].ID != "b" {
			t.Errorf("순서 = %v, 기대값 [a c b]", mountIDs(mounts))
		}
	})

	// 부정 케이스: 이미 맨 아래에 있는 항목은 더 아래로 이동할 수 없다.
	Scenario(t, "GIVEN 이미 맨 아래에 있는 항목 WHEN 아래로 이동 시도 THEN 아무 변화 없이 실패를 반환한다 (경계 케이스)", func(t *testing.T) {
		mounts := []engine.Mount{{ID: "a"}, {ID: "b"}}
		ok := swapMountsDown(mounts, 1)
		if ok {
			t.Errorf("맨 아래 항목은 아래로 이동할 수 없어야 함")
		}
	})

	// 부정 케이스: 빈 목록에서의 호출도 panic 없이 안전해야 한다.
	Scenario(t, "GIVEN 빈 목록 WHEN 위/아래 이동 시도 THEN panic 없이 실패를 반환한다 (부정 케이스)", func(t *testing.T) {
		var empty []engine.Mount
		if swapMountsUp(empty, 0) || swapMountsDown(empty, 0) {
			t.Errorf("빈 목록에서는 항상 실패해야 함")
		}
	})
}

func TestSwapRemotesUpDown(t *testing.T) {
	Scenario(t, "GIVEN 원본 리모트 목록의 중간 항목 WHEN 위로 이동 THEN 바로 앞 항목과 자리가 바뀐다", func(t *testing.T) {
		remotes := []engine.Remote{{Name: "a"}, {Name: "b"}, {Name: "c"}}
		ok := swapRemotesUp(remotes, 2)
		if !ok {
			t.Fatalf("이동이 성공했어야 함")
		}
		if remotes[1].Name != "c" || remotes[2].Name != "b" {
			t.Errorf("순서 = %v, 기대값 [a c b]", remoteNames(remotes))
		}
	})
}

func TestIndexOfMountAndRemote(t *testing.T) {
	Scenario(t, "GIVEN ID가 일치하는 마운트가 목록에 있음 WHEN 인덱스 조회 THEN 그 위치를 반환한다", func(t *testing.T) {
		mounts := []engine.Mount{{ID: "a"}, {ID: "b"}}
		if got := indexOfMount(mounts, "b"); got != 1 {
			t.Errorf("got %d, 기대값 1", got)
		}
	})

	// 부정 케이스: 존재하지 않는 ID로 조회.
	Scenario(t, "GIVEN 목록에 없는 ID WHEN 인덱스 조회 THEN -1을 반환한다 (부정 케이스)", func(t *testing.T) {
		mounts := []engine.Mount{{ID: "a"}}
		if got := indexOfMount(mounts, "does-not-exist"); got != -1 {
			t.Errorf("got %d, 기대값 -1", got)
		}
	})
}

func mountIDs(mounts []engine.Mount) []string {
	ids := make([]string, len(mounts))
	for i, m := range mounts {
		ids[i] = m.ID
	}
	return ids
}

func remoteNames(remotes []engine.Remote) []string {
	names := make([]string, len(remotes))
	for i, r := range remotes {
		names[i] = r.Name
	}
	return names
}
