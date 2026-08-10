package main

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed assets/icon.png
var iconPNG []byte

// appIcon is the app/window/tray icon, embedded into the binary so the
// exe never depends on a loose file next to it. Used to replace the
// generic Fyne/Go icon Windows was showing in the title bar and taskbar.
var appIcon = fyne.NewStaticResource("icon.png", iconPNG)
