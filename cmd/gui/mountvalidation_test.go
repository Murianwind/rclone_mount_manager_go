package main

import (
	"testing"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func TestValidateMount(t *testing.T) {
	existing := []engine.Mount{
		{ID: "a", Remote: "PLEX", RemotePath: "KODI", Drive: "E:"},
		{ID: "b", Remote: "gds", RemotePath: "VIDEO", Drive: "G:"},
	}

	Scenario(t, "GIVEN 다른 마운트와 드라이브 문자가 겹침 WHEN 검증 THEN 오류 메시지를 반환한다", func(t *testing.T) {
		m := engine.Mount{ID: "new", Remote: "onedrive", RemotePath: "x", Drive: "E:"}
		if got := validateMount(m, existing); got == "" {
			t.Errorf("드라이브 중복인데 오류가 없음")
		}
	})

	Scenario(t, "GIVEN 다른 마운트와 리모트+서브경로가 완전히 같음 WHEN 검증 THEN 오류 메시지를 반환한다", func(t *testing.T) {
		m := engine.Mount{ID: "new", Remote: "PLEX", RemotePath: "KODI", Drive: "Z:"}
		if got := validateMount(m, existing); got == "" {
			t.Errorf("리모트/경로 중복인데 오류가 없음")
		}
	})

	Scenario(t, "GIVEN 겹치는 게 전혀 없음 WHEN 검증 THEN 빈 문자열(문제 없음)을 반환한다", func(t *testing.T) {
		m := engine.Mount{ID: "new", Remote: "onedrive", RemotePath: "x", Drive: "Z:"}
		if got := validateMount(m, existing); got != "" {
			t.Errorf("문제 없어야 하는데 %q", got)
		}
	})

	// 부정/경계 케이스: 자기 자신(편집 중인 마운트)은 비교 대상에서 제외돼야 함.
	Scenario(t, "GIVEN 편집 중인 마운트가 자기 자신의 예전 값과 비교됨 WHEN 검증 THEN 자기 자신은 충돌로 안 잡는다 (경계 케이스)", func(t *testing.T) {
		m := engine.Mount{ID: "a", Remote: "PLEX", RemotePath: "KODI", Drive: "E:"} // 기존 "a"를 그대로 저장(편집 후 변경 없음)
		if got := validateMount(m, existing); got != "" {
			t.Errorf("자기 자신과는 충돌로 잡히면 안 되는데 %q", got)
		}
	})

	// 부정 케이스: 드라이브 문자가 비어있으면(자동 배정) 다른 빈 드라이브와도 충돌로 안 잡음.
	Scenario(t, "GIVEN 드라이브 문자가 둘 다 비어있음(자동) WHEN 검증 THEN 드라이브 중복으로 안 잡는다 (부정 케이스)", func(t *testing.T) {
		existingAuto := []engine.Mount{{ID: "a", Remote: "PLEX", RemotePath: "KODI", Drive: ""}}
		m := engine.Mount{ID: "new", Remote: "gds", RemotePath: "x", Drive: ""}
		if got := validateMount(m, existingAuto); got != "" {
			t.Errorf("빈 드라이브끼리는 충돌 아니어야 하는데 %q", got)
		}
	})
}
