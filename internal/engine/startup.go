package engine

import "os"

// GetCurrentExePath returns the current executable's path in the quoted
// form used for the registry Run-key value, mirroring
// get_current_exe_path().
//
// Go always compiles to a single native executable, so unlike the Python
// version — which had a separate "pythonw script" fallback for
// unfrozen/dev runs — there is only one code path here.
func GetCurrentExePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	// Plain quote-wrap, matching Python's f'"{sys.executable}"' — no
	// escaping of the backslashes in a Windows path.
	return `"` + exe + `"`, nil
}

// Indirected through vars (rather than called directly) so tests can
// substitute the platform-specific registry backend and exercise
// CheckAndFixStartup's branching logic without touching a real registry —
// the same thing the Python tests do via unittest.mock.patch on the
// module-level functions.
var (
	startupPathFn    = GetStartupPath
	setStartupFn     = SetStartup
	currentExePathFn = GetCurrentExePath
)

// CheckAndFixStartup re-registers the startup entry if one is present but
// points at a path different from the currently running executable (e.g.
// after the user moved/updated the exe). Returns true if a
// re-registration was performed. Mirrors check_and_fix_startup().
func CheckAndFixStartup() (bool, error) {
	registered := startupPathFn()
	if registered == "" {
		return false, nil // not registered -> nothing to do
	}

	current, err := currentExePathFn()
	if err != nil {
		return false, err
	}
	if registered == current {
		return false, nil // already correct
	}

	if err := setStartupFn(true); err != nil {
		return false, err
	}
	return true, nil
}
