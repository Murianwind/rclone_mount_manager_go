//go:build !windows

package engine

import "errors"

// ErrRegistryUnavailable is returned by SetStartup on platforms with no
// Windows registry. This mirrors the Python version's
// "try: import winreg / except ImportError: winreg = None" fallback,
// which exists specifically so the app is buildable and testable on
// non-Windows machines (e.g. this CI's Linux runners).
var ErrRegistryUnavailable = errors.New("windows registry not available on this platform")

func IsStartupEnabled() bool { return false }

func GetStartupPath() string { return "" }

func SetStartup(enable bool) error { return ErrRegistryUnavailable }
