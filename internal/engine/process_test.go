//go:build !windows

package engine

import (
	"errors"
	"os/exec"
	"testing"
)

func TestConfigureBackgroundProcess_DoesNotPanicOnThisPlatform(t *testing.T) {
	cmd := exec.Command("echo", "hi")
	ConfigureBackgroundProcess(cmd) // no-op here; must not panic or crash
	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error running command: %v", err)
	}
}

func TestSignalGracefulStop_UnsupportedOnThisPlatform(t *testing.T) {
	err := SignalGracefulStop(1)
	if !errors.Is(err, ErrGracefulStopUnsupported) {
		t.Errorf("expected ErrGracefulStopUnsupported, got %v", err)
	}
}
