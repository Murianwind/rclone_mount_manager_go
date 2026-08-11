package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// tapCatcher is an invisible, fully-transparent widget that only reacts
// to taps. Placed as the bottom-most layer of the window content, it
// catches clicks that land on empty space not already claimed by a more
// specific widget (buttons, entries, table cells all consume their own
// taps first — Fyne dispatches to the topmost/most specific object under
// the cursor). Used to clear the table's row selection when the user
// clicks outside the table.
type tapCatcher struct {
	widget.BaseWidget
	onTapped func()
}

func newTapCatcher(onTapped func()) *tapCatcher {
	t := &tapCatcher{onTapped: onTapped}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tapCatcher) Tapped(_ *fyne.PointEvent) {
	if t.onTapped != nil {
		t.onTapped()
	}
}

func (t *tapCatcher) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}
