package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	fynetest "fyne.io/fyne/v2/test"
)

// TestVerticalGuardLayout_UsesContentMinSizeNotScrollMinSize is a direct
// regression test for the actual bug reported: verticalGuardLayout was
// asking the *Scroll wrapper* how much height it wanted, and a Scroll's
// own MinSize() is a small fixed value (proven below) regardless of its
// content — so a perfectly normal-sized window still gave "top" almost no
// space, hiding rows and adding stray scrollbars.
func TestVerticalGuardLayout_UsesContentMinSizeNotScrollMinSize(t *testing.T) {
	fynetest.NewApp()

	// 먼저 버그의 전제 자체를 확인해둔다: Scroll의 MinSize는 내용물의
	// MinSize를 반영하지 않는다.
	probe := canvas.NewRectangle(nil)
	probe.SetMinSize(fyne.NewSize(0, 140))
	probeScroll := container.NewVScroll(probe)
	if probeScroll.MinSize().Height == probe.MinSize().Height {
		t.Fatalf("전제 확인 실패: 이 Fyne 버전에서는 Scroll.MinSize()가 내용물 크기를 그대로 반영하는 것 같음 — 테스트 가정을 다시 봐야 함")
	}

	Scenario(t, "GIVEN 상단/하단이 Scroll로 감싸져 있고 창이 충분히 큼 WHEN 레이아웃 계산 THEN Scroll의 작은 최소값이 아니라 내용물이 실제로 필요로 하는 높이를 받는다 (회귀 테스트)", func(t *testing.T) {
		topContent := canvas.NewRectangle(nil)
		topContent.SetMinSize(fyne.NewSize(0, 140))
		topScroll := container.NewVScroll(topContent)

		bottomContent := canvas.NewRectangle(nil)
		bottomContent.SetMinSize(fyne.NewSize(0, 50))
		bottomScroll := container.NewVScroll(bottomContent)

		center := canvas.NewRectangle(nil)

		l := &verticalGuardLayout{
			top: topScroll, center: center, bottom: bottomScroll,
			topContent: topContent, bottomContent: bottomContent,
			minTop: 32, minBottom: 32,
		}

		l.Layout(nil, fyne.NewSize(300, 500))

		if topScroll.Size().Height != 140 {
			t.Errorf("topScroll 높이 = %v, 기대값 140 (내용물이 실제로 필요로 하는 높이)", topScroll.Size().Height)
		}
		if bottomScroll.Size().Height != 50 {
			t.Errorf("bottomScroll 높이 = %v, 기대값 50", bottomScroll.Size().Height)
		}
		wantCenter := float32(500 - 140 - 50)
		if center.Size().Height != wantCenter {
			t.Errorf("center 높이 = %v, 기대값 %v", center.Size().Height, wantCenter)
		}
	})

	Scenario(t, "GIVEN 창이 상단+하단이 필요로 하는 높이보다 작음 WHEN 레이아웃 계산 THEN 상단/하단이 최소치로 줄고 center는 절대 음수가 안 된다 (경계 케이스)", func(t *testing.T) {
		topContent := canvas.NewRectangle(nil)
		topContent.SetMinSize(fyne.NewSize(0, 140))
		topScroll := container.NewVScroll(topContent)

		bottomContent := canvas.NewRectangle(nil)
		bottomContent.SetMinSize(fyne.NewSize(0, 50))
		bottomScroll := container.NewVScroll(bottomContent)

		center := canvas.NewRectangle(nil)

		l := &verticalGuardLayout{
			top: topScroll, center: center, bottom: bottomScroll,
			topContent: topContent, bottomContent: bottomContent,
			minTop: 32, minBottom: 32,
		}

		l.Layout(nil, fyne.NewSize(300, 100)) // 100 < 140+50

		if center.Size().Height < 0 {
			t.Fatalf("center 높이가 음수(%v)면 안 됨", center.Size().Height)
		}
		if topScroll.Size().Height != 32 || bottomScroll.Size().Height != 32 {
			t.Errorf("공간이 부족하면 상단/하단이 최소치(32)까지 줄어야 하는데 top=%v bottom=%v",
				topScroll.Size().Height, bottomScroll.Size().Height)
		}
	})
}
