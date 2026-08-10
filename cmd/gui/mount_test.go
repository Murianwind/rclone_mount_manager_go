package main

import (
	"errors"
	"testing"
)

// ── keywords ──

func givenExitError(msg string) error {
	if msg == "" {
		return nil
	}
	return errors.New(msg)
}

func whenDecidingToReportFailure(exitErr error, stoppedByUs bool) bool {
	return shouldReportMountFailure(exitErr, stoppedByUs)
}

func thenDecisionIs(t *testing.T, got, want bool) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestShouldReportMountFailure(t *testing.T) {
	Scenario(t, "GIVEN a clean exit that we did not request WHEN deciding whether to alert THEN it does not alert (nothing went wrong)", func(t *testing.T) {
		got := whenDecidingToReportFailure(givenExitError(""), false)
		thenDecisionIs(t, got, false)
	})

	Scenario(t, "GIVEN a clean exit that we requested (normal unmount) WHEN deciding whether to alert THEN it does not alert", func(t *testing.T) {
		got := whenDecidingToReportFailure(givenExitError(""), true)
		thenDecisionIs(t, got, false)
	})

	Scenario(t, "GIVEN a non-zero exit that WE requested (unmount often exits non-zero) WHEN deciding whether to alert THEN it does not alert — this is expected, not a failure", func(t *testing.T) {
		got := whenDecidingToReportFailure(givenExitError("exit status 1"), true)
		thenDecisionIs(t, got, false)
	})

	// This is the real negative/failure case the feature exists for: an
	// rclone process that dies on its own, unasked — e.g. WinFsp
	// "mountpoint already exists" or a network/auth failure reaching the
	// remote.
	Scenario(t, "GIVEN a non-zero exit that we did NOT request WHEN deciding whether to alert THEN it alerts — this is a genuine mount failure", func(t *testing.T) {
		got := whenDecidingToReportFailure(givenExitError("exit status 1"), false)
		thenDecisionIs(t, got, true)
	})
}

func TestFormatRcloneVersionLabel(t *testing.T) {
	Scenario(t, "GIVEN rclone version ran successfully WHEN formatted THEN the parsed version is shown", func(t *testing.T) {
		got := formatRcloneVersionLabel("rclone v1.68.2\n- os/version: windows", nil)
		if got != "rclone v1.68.2" {
			t.Errorf("got %q, want %q", got, "rclone v1.68.2")
		}
	})

	// Negative case: rclone.exe missing, permission denied, or crashed.
	Scenario(t, "GIVEN the rclone process itself failed to run WHEN formatted THEN it falls back to 'unknown' rather than showing stale/blank text", func(t *testing.T) {
		got := formatRcloneVersionLabel("", errors.New("exec: \"rclone.exe\": file does not exist"))
		if got != "v알 수 없음" {
			t.Errorf("got %q, want %q", got, "v알 수 없음")
		}
	})

	Scenario(t, "GIVEN the process succeeded but produced unparsable output WHEN formatted THEN it falls back to 'unknown'", func(t *testing.T) {
		got := formatRcloneVersionLabel("not a version string", nil)
		if got != "v알 수 없음" {
			t.Errorf("got %q, want %q", got, "v알 수 없음")
		}
	})
}
