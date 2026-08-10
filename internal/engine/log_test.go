package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotatingLog_RotatesWhenOverMaxLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "RcloneManager.log")
	if err := os.WriteFile(path, []byte("l1\nl2\nl3\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	l := RotatingLog{Path: path, MaxLines: 3}
	if err := l.Write("INFO", "새 라인"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines after rotation, got %d: %v", len(lines), lines)
	}
	// oldest line (l1) must have been dropped, newest line must be present
	if lines[0] != "l2" {
		t.Errorf("expected oldest surviving line to be l2, got %q", lines[0])
	}
	if !strings.Contains(lines[2], "새 라인") {
		t.Errorf("expected newest line to contain the new message, got %q", lines[2])
	}
}

func TestRotatingLog_FailureIsNonFatal(t *testing.T) {
	// Path is a directory, not a file -> WriteFile must fail, but the call
	// must return cleanly (no panic), matching write_log's "swallow errors"
	// intent — the caller is expected to ignore the returned error.
	dir := t.TempDir()
	l := RotatingLog{Path: dir, MaxLines: 10}

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Write panicked: %v", r)
			}
		}()
		if err := l.Write("ERROR", "실패해도 괜찮음"); err == nil {
			t.Errorf("expected an error when writing to a directory path")
		}
	}()
}
