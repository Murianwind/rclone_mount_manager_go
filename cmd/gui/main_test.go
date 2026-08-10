package main

import "testing"

func TestSavedOr(t *testing.T) {
	Scenario(t, "GIVEN 저장된 창 크기가 있음(양수) WHEN 크기 결정 THEN 저장된 값을 사용한다", func(t *testing.T) {
		got := savedOr(900, defaultWindowWidth)
		if got != 900 {
			t.Errorf("got %v, 기대값 900", got)
		}
	})

	// 부정/경계 케이스: 저장된 값이 0(한 번도 저장된 적 없음)인 최초 실행 상황.
	Scenario(t, "GIVEN 저장된 창 크기가 0(최초 실행, 저장된 적 없음) WHEN 크기 결정 THEN 기본값을 사용한다 (경계 케이스)", func(t *testing.T) {
		got := savedOr(0, defaultWindowWidth)
		if got != defaultWindowWidth {
			t.Errorf("got %v, 기대값 기본값 %v", got, defaultWindowWidth)
		}
	})

	// 부정 케이스: 음수처럼 비정상적인 저장값이 들어온 경우에도 기본값으로
	// 폴백해야 한다 (설정 파일이 손상/수동 편집된 경우 대비).
	Scenario(t, "GIVEN 저장된 창 크기가 음수(비정상 값) WHEN 크기 결정 THEN 기본값으로 폴백한다 (부정 케이스)", func(t *testing.T) {
		got := savedOr(-100, defaultWindowWidth)
		if got != defaultWindowWidth {
			t.Errorf("got %v, 기대값 기본값 %v", got, defaultWindowWidth)
		}
	})
}
