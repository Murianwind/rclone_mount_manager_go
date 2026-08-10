package main

import (
	"testing"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func TestTrayShortLabel(t *testing.T) {
	if got := trayShortLabel(engine.Mount{Remote: "gds", Drive: "G:"}); got != "G:" {
		t.Errorf("trayShortLabel with a drive set = %q, want %q", got, "G:")
	}
	if got := trayShortLabel(engine.Mount{Remote: "gds"}); got != "gds" {
		t.Errorf("trayShortLabel with no drive = %q, want %q (fallback to remote)", got, "gds")
	}
}

func TestTrayDisplayLabel(t *testing.T) {
	m := engine.Mount{Remote: "PLEX", RemotePath: "KODI", Drive: "E:"}

	running := trayDisplayLabel(m, true)
	if running != "■  E:  (PLEX:KODI)" {
		t.Errorf("running label = %q, want %q", running, "■  E:  (PLEX:KODI)")
	}

	stopped := trayDisplayLabel(m, false)
	if stopped != "▶  E:  (PLEX:KODI)" {
		t.Errorf("stopped label = %q, want %q", stopped, "▶  E:  (PLEX:KODI)")
	}
}

func TestTrayDisplayLabel_NoRemotePath(t *testing.T) {
	// A remote with no sub-path (e.g. "nas:") shouldn't show a trailing
	// bare colon in the parenthetical.
	m := engine.Mount{Remote: "nas"}
	got := trayDisplayLabel(m, false)
	if got != "▶  nas  (nas)" {
		t.Errorf("got %q, want %q", got, "▶  nas  (nas)")
	}
}
