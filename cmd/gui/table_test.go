package main

import "testing"

func TestDisplayDrive(t *testing.T) {
	if got := displayDrive(""); got != "(자동)" {
		t.Errorf("displayDrive(\"\") = %q, want %q", got, "(자동)")
	}
	if got := displayDrive("  "); got != "(자동)" {
		t.Errorf("displayDrive(\"  \") = %q, want %q", got, "(자동)")
	}
	if got := displayDrive("E:"); got != "E:" {
		t.Errorf("displayDrive(\"E:\") = %q, want %q", got, "E:")
	}
}

func TestStatusLabel(t *testing.T) {
	if got := statusLabel(true); got != "연결됨" {
		t.Errorf("statusLabel(true) = %q, want %q", got, "연결됨")
	}
	if got := statusLabel(false); got != "해제됨" {
		t.Errorf("statusLabel(false) = %q, want %q", got, "해제됨")
	}
}

func TestToggleLabel(t *testing.T) {
	if got := toggleLabel(true); got != "해제" {
		t.Errorf("toggleLabel(true) = %q, want %q", got, "해제")
	}
	if got := toggleLabel(false); got != "마운트" {
		t.Errorf("toggleLabel(false) = %q, want %q", got, "마운트")
	}
}
