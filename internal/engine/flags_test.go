package engine

import "testing"

func TestNormalizeFlags_Empty(t *testing.T) {
	if got := NormalizeFlags(""); got != "" {
		t.Errorf("NormalizeFlags(\"\") = %q, want \"\"", got)
	}
	if got := NormalizeFlags("   "); got != "" {
		t.Errorf("NormalizeFlags(\"   \") = %q, want \"\"", got)
	}
}

func TestNormalizeFlags_NoValueFlagsPreserved(t *testing.T) {
	got := NormalizeFlags("--links;--fast-list")
	want := "--links;--fast-list"
	if got != want {
		t.Errorf("NormalizeFlags = %q, want %q", got, want)
	}
}

func TestNormalizeFlags_SpaceSeparatedBecomesEquals(t *testing.T) {
	got := NormalizeFlags("--read-only; --bwlimit 10M")
	want := "--read-only;--bwlimit=10M"
	if got != want {
		t.Errorf("NormalizeFlags = %q, want %q", got, want)
	}
}

func TestNormalizeFlags_MultipleFlagsInOneChunk(t *testing.T) {
	got := NormalizeFlags("--no-modtime --no-checksum")
	want := "--no-modtime;--no-checksum"
	if got != want {
		t.Errorf("NormalizeFlags = %q, want %q", got, want)
	}
}

func TestNormalizeFlags_NewlineSeparated(t *testing.T) {
	got := NormalizeFlags("--flag value\n--flag2=value2")
	want := "--flag=value;--flag2=value2"
	if got != want {
		t.Errorf("NormalizeFlags = %q, want %q", got, want)
	}
}
