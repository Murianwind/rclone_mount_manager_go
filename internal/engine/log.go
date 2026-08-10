package engine

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// RotatingLog appends timestamped lines to a log file, keeping at most
// MaxLines lines (oldest lines are dropped first). Direct port of
// write_log()/LOG_MAX_LINES.
type RotatingLog struct {
	Path     string
	MaxLines int
}

// Write appends one "TIMESTAMP LEVEL  message" line and rotates the file
// if it now exceeds MaxLines. Unlike the Python version (which silently
// swallows all errors so a logging failure never crashes the caller), this
// returns the error so callers can decide — but a logging failure here is
// never meant to be fatal, so callers should generally ignore it, e.g.
// "_ = myLog.Write(...)".
func (l RotatingLog) Write(level, message string) error {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("%s %-5s %s\n", timestamp, level, message)

	var lines []string
	if data, err := os.ReadFile(l.Path); err == nil {
		lines = strings.SplitAfter(string(data), "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
	}

	lines = append(lines, line)
	if l.MaxLines > 0 && len(lines) > l.MaxLines {
		lines = lines[len(lines)-l.MaxLines:]
	}

	return os.WriteFile(l.Path, []byte(strings.Join(lines, "")), 0o644)
}
