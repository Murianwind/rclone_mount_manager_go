package main

import (
	"testing"

	"fyne.io/fyne/v2"
)

func TestClampToMinWindowSize(t *testing.T) {
	Scenario(t, "GIVEN 현재 크기가 최소치보다 큼 WHEN 판정 THEN 리사이즈가 필요 없다고 판단한다", func(t *testing.T) {
		_, need := clampToMinWindowSize(fyne.NewSize(900, 600), 700, 340)
		if need {
			t.Errorf("이미 충분히 큰데 리사이즈가 필요하다고 나옴")
		}
	})

	Scenario(t, "GIVEN 폭만 최소치보다 작음 WHEN 판정 THEN 폭만 최소치로 올리고 높이는 그대로 둔다", func(t *testing.T) {
		target, need := clampToMinWindowSize(fyne.NewSize(500, 600), 700, 340)
		if !need {
			t.Fatalf("폭이 모자라면 리사이즈가 필요해야 함")
		}
		if target.Width != 700 || target.Height != 600 {
			t.Errorf("got %v, 기대값 {700 600}", target)
		}
	})

	Scenario(t, "GIVEN 높이만 최소치보다 작음 WHEN 판정 THEN 높이만 최소치로 올린다 (실제 겪은 버그 시나리오)", func(t *testing.T) {
		target, need := clampToMinWindowSize(fyne.NewSize(900, 150), 700, 340)
		if !need {
			t.Fatalf("높이가 모자라면 리사이즈가 필요해야 함")
		}
		if target.Width != 900 || target.Height != 340 {
			t.Errorf("got %v, 기대값 {900 340}", target)
		}
	})

	// 부정/경계 케이스: 정확히 최소치와 같음 — 리사이즈 불필요.
	Scenario(t, "GIVEN 현재 크기가 최소치와 정확히 같음 WHEN 판정 THEN 리사이즈가 필요 없다 (경계 케이스)", func(t *testing.T) {
		_, need := clampToMinWindowSize(fyne.NewSize(700, 340), 700, 340)
		if need {
			t.Errorf("최소치와 같으면 리사이즈가 필요 없어야 함")
		}
	})

	Scenario(t, "GIVEN 폭·높이 둘 다 최소치보다 작음 WHEN 판정 THEN 둘 다 최소치로 올린다", func(t *testing.T) {
		target, need := clampToMinWindowSize(fyne.NewSize(400, 100), 700, 340)
		if !need {
			t.Fatalf("둘 다 모자라면 리사이즈가 필요해야 함")
		}
		if target.Width != 700 || target.Height != 340 {
			t.Errorf("got %v, 기대값 {700 340}", target)
		}
	})
}
