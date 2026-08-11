package main

import "fyne.io/fyne/v2"

// verticalGuardLayout is a hand-rolled Border-style layout: top and
// bottom get their natural (wanted) height when there's room, but once
// total space is tight, THEY shrink first (down to minTop/minBottom,
// expected to be wrapped in a Scroll so shrinking doesn't overlap their
// own children) — guaranteeing center (the mount table) is never squeezed
// to zero or negative height.
//
// This replaces layout.NewBorderLayout for the window root specifically
// because Border always reserves top/bottom's full natural height no
// matter how little space is available, which is what let the table get
// crushed when the window was resized short.
type verticalGuardLayout struct {
	top, center, bottom fyne.CanvasObject
	minTop, minBottom   float32
}

func (l *verticalGuardLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	w := l.center.MinSize().Width
	if tw := l.top.MinSize().Width; tw > w {
		w = tw
	}
	if bw := l.bottom.MinSize().Width; bw > w {
		w = bw
	}
	return fyne.NewSize(w, l.minTop+l.minBottom+l.center.MinSize().Height)
}

func (l *verticalGuardLayout) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	topH, centerH, bottomH := computeVerticalSplit(
		size.Height, l.top.MinSize().Height, l.bottom.MinSize().Height, l.minTop, l.minBottom)

	l.top.Resize(fyne.NewSize(size.Width, topH))
	l.top.Move(fyne.NewPos(0, 0))

	l.center.Resize(fyne.NewSize(size.Width, centerH))
	l.center.Move(fyne.NewPos(0, topH))

	l.bottom.Resize(fyne.NewSize(size.Width, bottomH))
	l.bottom.Move(fyne.NewPos(0, topH+centerH))
}

// computeVerticalSplit decides how much height to give top/center/bottom.
// When there's enough room, top and bottom get exactly what they asked
// for (topWant/bottomWant) and center gets whatever's left. When there
// isn't, top and bottom yield down to minTop/minBottom (scaling those
// down further, proportionally, if even that doesn't fit) so center's
// share is never negative.
//
// Pulled out as a pure function — no fyne.CanvasObject involved — so this
// arithmetic can be tested without a running window; see
// verticallayout_test.go.
func computeVerticalSplit(total, topWant, bottomWant, minTop, minBottom float32) (top, center, bottom float32) {
	if total >= topWant+bottomWant {
		return topWant, total - topWant - bottomWant, bottomWant
	}

	top, bottom = minTop, minBottom
	if top+bottom > total {
		if total <= 0 {
			return 0, 0, 0
		}
		scale := total / (top + bottom)
		top *= scale
		bottom *= scale
	}

	center = total - top - bottom
	if center < 0 {
		center = 0
	}
	return top, center, bottom
}
