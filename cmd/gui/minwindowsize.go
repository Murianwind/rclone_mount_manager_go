package main

import (
	"time"

	"fyne.io/fyne/v2"
)

// minWindowHeight is the smallest height the window is allowed to settle
// at — enough for the header/path/checkbox rows, the table header + a
// couple of rows, and the bottom button bar without any of them being
// hidden or overlapping.
const minWindowHeight = 340

// enforceMinWindowSize keeps the window from staying smaller than
// tableContentWidth+20 × minWindowHeight.
//
// Fyne (this version, desktop/glfw driver) has no window-resize event and
// no "set minimum size" API at all — window.Resize() only affects
// programmatic resizes, not interactive dragging. So instead of trying to
// *prevent* a too-small resize, this polls the canvas size and snaps it
// back up the moment it dips below the floor. There's a brief visual
// flicker during an active drag, but the window can never actually remain
// smaller than the floor — which is the only user-visible thing that
// matters.
func enforceMinWindowSize(win fyne.Window) {
	minW := float32(tableContentWidth + 20)
	go func() {
		ticker := time.NewTicker(150 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			target, need := clampToMinWindowSize(win.Canvas().Size(), minW, minWindowHeight)
			if need {
				fyne.Do(func() { win.Resize(target) })
			}
		}
	}()
}

// clampToMinWindowSize decides whether current is below the floor and, if
// so, what size to snap back up to. Pulled out as a pure function for
// testing — see minwindowsize_test.go.
func clampToMinWindowSize(current fyne.Size, minW, minH float32) (target fyne.Size, needsResize bool) {
	w, h := current.Width, current.Height
	if w < minW {
		w = minW
		needsResize = true
	}
	if h < minH {
		h = minH
		needsResize = true
	}
	return fyne.NewSize(w, h), needsResize
}
