package main

import (
	"errors"
	"testing"
)

func TestShouldReportMountFailure(t *testing.T) {
	cases := []struct {
		name        string
		exitErr     error
		stoppedByUs bool
		want        bool
	}{
		{"clean exit, not stopped by us", nil, false, false},
		{"clean exit, stopped by us", nil, true, false},
		{"error exit, we asked it to stop (normal unmount)", errors.New("exit status 1"), true, false},
		{"error exit, unexpected (real failure)", errors.New("exit status 1"), false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := shouldReportMountFailure(c.exitErr, c.stoppedByUs)
			if got != c.want {
				t.Errorf("shouldReportMountFailure(%v, %v) = %v, want %v", c.exitErr, c.stoppedByUs, got, c.want)
			}
		})
	}
}

func TestFormatRcloneVersionLabel(t *testing.T) {
	cases := []struct {
		name   string
		output string
		err    error
		want   string
	}{
		{"success", "rclone v1.68.2\n- os/version: windows", nil, "rclone v1.68.2"},
		{"process error", "", errors.New("exit status 1"), "v알 수 없음"},
		{"unparsable output", "not a version string", nil, "v알 수 없음"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := formatRcloneVersionLabel(c.output, c.err)
			if got != c.want {
				t.Errorf("formatRcloneVersionLabel(%q, %v) = %q, want %q", c.output, c.err, got, c.want)
			}
		})
	}
}
