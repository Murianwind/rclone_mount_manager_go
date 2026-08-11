package main

import "testing"

func TestComputeVerticalSplit(t *testing.T) {
	Scenario(t, "GIVEN 창 높이가 상단+하단이 원하는 만큼보다 충분히 큼 WHEN 배분 계산 THEN 상단·하단은 원하는 만큼 받고 표가 나머지 전부를 받는다", func(t *testing.T) {
		top, center, bottom := computeVerticalSplit(500, 100, 50, 40, 20)
		if top != 100 || bottom != 50 {
			t.Errorf("top=%v bottom=%v, 기대값 100/50 (원하는 만큼 그대로)", top, bottom)
		}
		if center != 350 {
			t.Errorf("center = %v, 기대값 350 (500-100-50)", center)
		}
	})

	Scenario(t, "GIVEN 창 높이가 상단+하단이 원하는 만큼과 정확히 같음 WHEN 배분 계산 THEN 표는 0을 받지만 음수는 아니다 (경계 케이스)", func(t *testing.T) {
		_, center, _ := computeVerticalSplit(150, 100, 50, 40, 20)
		if center != 0 {
			t.Errorf("center = %v, 기대값 0", center)
		}
	})

	// 핵심 회귀 테스트: 이게 바로 사용자가 겪은 버그 — 공간이 모자라면
	// 표가 음수 높이를 받아 찌그러지던 문제.
	Scenario(t, "GIVEN 창 높이가 상단+하단이 원하는 만큼보다 작음(창을 세로로 많이 줄임) WHEN 배분 계산 THEN 상단·하단이 최소치까지 양보하고 표는 절대 음수를 받지 않는다 (부정 케이스/회귀 테스트)", func(t *testing.T) {
		top, center, bottom := computeVerticalSplit(80, 100, 50, 40, 20)
		if top != 40 || bottom != 20 {
			t.Errorf("top=%v bottom=%v, 기대값 최소치 40/20", top, bottom)
		}
		if center < 0 {
			t.Fatalf("center가 음수(%v)면 안 됨 — 이게 바로 표가 찌그러지던 버그", center)
		}
		if center != 20 {
			t.Errorf("center = %v, 기대값 20 (80-40-20)", center)
		}
	})

	// 부정 케이스: 최소치를 다 줘도 모자랄 만큼 극단적으로 작은 창.
	Scenario(t, "GIVEN 최소치(minTop+minBottom)조차 못 채울 만큼 창이 극단적으로 작음 WHEN 배분 계산 THEN 상단·하단을 비율대로 줄여서라도 합이 전체 높이를 넘지 않게 하고, 표는 0으로 바닥을 친다 (부정 케이스)", func(t *testing.T) {
		top, center, bottom := computeVerticalSplit(30, 100, 50, 40, 20)
		if top+bottom > 30 {
			t.Errorf("top+bottom = %v, 창 높이(30)를 넘으면 안 됨", top+bottom)
		}
		if center != 0 {
			t.Errorf("center = %v, 기대값 0", center)
		}
		// 비율(40:20 = 2:1)은 유지돼야 한다.
		if top != 20 || bottom != 10 {
			t.Errorf("top=%v bottom=%v, 기대값 20/10 (2:1 비율로 30에 맞춰 축소)", top, bottom)
		}
	})

	// 부정 케이스: 창 높이가 0 이하인 극단 상황 — panic이나 음수 반환이 없어야 함.
	Scenario(t, "GIVEN 창 높이가 0 WHEN 배분 계산 THEN panic 없이 전부 0을 반환한다 (부정 케이스)", func(t *testing.T) {
		top, center, bottom := computeVerticalSplit(0, 100, 50, 40, 20)
		if top != 0 || center != 0 || bottom != 0 {
			t.Errorf("top=%v center=%v bottom=%v, 전부 0이어야 함", top, center, bottom)
		}
	})
}
