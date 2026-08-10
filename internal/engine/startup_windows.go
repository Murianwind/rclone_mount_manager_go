//go:build windows

package engine

import "golang.org/x/sys/windows/registry"

const (
	startupValueName = "RcloneManager"
	runKeyPath       = `Software\Microsoft\Windows\CurrentVersion\Run`
)

// IsStartupEnabled reports whether the app is registered to launch at
// Windows login. Mirrors is_startup_enabled().
func IsStartupEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	_, _, err = key.GetStringValue(startupValueName)
	return err == nil
}

// GetStartupPath returns the path currently registered in the Run key, or
// "" if not registered / on any error. Mirrors get_startup_path().
func GetStartupPath() string {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer key.Close()
	v, _, err := key.GetStringValue(startupValueName)
	if err != nil {
		return ""
	}
	return v
}

// SetStartup registers (enable=true) or removes (enable=false) the
// startup entry. Mirrors set_startup(): removing a non-existent value is
// not an error (the Python version silently ignores that case too).
func SetStartup(enable bool) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if !enable {
		_ = key.DeleteValue(startupValueName) // ignore "not found"
		return nil
	}

	path, err := GetCurrentExePath()
	if err != nil {
		return err
	}
	return key.SetStringValue(startupValueName, path)
}
