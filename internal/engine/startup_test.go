package engine

import (
	"errors"
	"strings"
	"testing"
)

// On this CI's Linux runners, the build-tag-selected implementation is the
// !windows stub (startup_other.go) — the direct analogue of the Python
// tests that patch `rclone_manager.winreg` to None. On an actual Windows
// build, startup_windows.go is compiled instead and these three would
// exercise the real (unavailable-registry-key) failure paths.

func TestIsStartupEnabled_NoRegistryOnThisPlatform(t *testing.T) {
	if IsStartupEnabled() {
		t.Errorf("expected false without a real Windows registry")
	}
}

func TestGetStartupPath_NoRegistryOnThisPlatform(t *testing.T) {
	if got := GetStartupPath(); got != "" {
		t.Errorf("GetStartupPath() = %q, want \"\"", got)
	}
}

func TestSetStartup_NoRegistryOnThisPlatform(t *testing.T) {
	if err := SetStartup(true); err == nil {
		t.Errorf("expected an error without a real Windows registry")
	}
}

func TestGetCurrentExePath_IsQuoted(t *testing.T) {
	path, err := GetCurrentExePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(path, `"`) || !strings.HasSuffix(path, `"`) {
		t.Errorf("expected a quote-wrapped path, got %q", path)
	}
}

// ── CheckAndFixStartup branching, mirroring scenarios 48-50 ──

func withStartupFns(t *testing.T, pathFn func() string, setFn func(bool) error, exeFn func() (string, error)) {
	t.Helper()
	origPath, origSet, origExe := startupPathFn, setStartupFn, currentExePathFn
	startupPathFn, setStartupFn, currentExePathFn = pathFn, setFn, exeFn
	t.Cleanup(func() {
		startupPathFn, setStartupFn, currentExePathFn = origPath, origSet, origExe
	})
}

func TestCheckAndFixStartup_NotRegistered(t *testing.T) {
	withStartupFns(t,
		func() string { return "" },
		func(bool) error { return nil },
		func() (string, error) { return "current", nil },
	)
	changed, err := CheckAndFixStartup()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Errorf("expected no change when not registered")
	}
}

func TestCheckAndFixStartup_PathMatches(t *testing.T) {
	withStartupFns(t,
		func() string { return "samepath" },
		func(bool) error { t.Fatalf("SetStartup should not be called when paths match"); return nil },
		func() (string, error) { return "samepath", nil },
	)
	changed, err := CheckAndFixStartup()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Errorf("expected no change when path already matches")
	}
}

func TestCheckAndFixStartup_PathMismatchReregisters(t *testing.T) {
	var setCalledWith *bool
	withStartupFns(t,
		func() string { return "old" },
		func(enable bool) error { setCalledWith = &enable; return nil },
		func() (string, error) { return "new", nil },
	)
	changed, err := CheckAndFixStartup()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Errorf("expected re-registration on path mismatch")
	}
	if setCalledWith == nil || *setCalledWith != true {
		t.Errorf("expected SetStartup to be called with true")
	}
}

func TestCheckAndFixStartup_SetStartupFailurePropagates(t *testing.T) {
	wantErr := errors.New("boom")
	withStartupFns(t,
		func() string { return "old" },
		func(bool) error { return wantErr },
		func() (string, error) { return "new", nil },
	)
	_, err := CheckAndFixStartup()
	if !errors.Is(err, wantErr) {
		t.Errorf("expected the underlying error to propagate, got %v", err)
	}
}
