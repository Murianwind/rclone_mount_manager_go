package engine

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// DownloadAppUpdate downloads a release asset zip (expected to contain
// RcloneManager.exe) and extracts the exe to
// destDir/RcloneManager_new.exe, ready for ApplyUpdate.
//
// This replaces the Python version's download_app_release(), which could
// only ever save the update file for the user to swap in manually — a
// running Windows exe can't be overwritten in place. ApplyUpdate below
// solves that with the rename trick instead.
func DownloadAppUpdate(client *http.Client, destDir, assetURL string) (newExePath string, err error) {
	if client == nil {
		client = defaultDownloadClient
	}

	data, err := httpGetBytes(client, assetURL)
	if err != nil {
		return "", err
	}
	exeData, err := extractFileFromZip(data, "RcloneManager.exe")
	if err != nil {
		return "", err
	}
	newExePath = filepath.Join(destDir, "RcloneManager_new.exe")
	if err := os.WriteFile(newExePath, exeData, 0o755); err != nil {
		return "", err
	}
	return newExePath, nil
}

// launchFn starts the (newly updated) exe. It's a package var so tests can
// substitute a no-op instead of actually spawning a process.
var launchFn = func(exePath string) error {
	cmd := exec.Command(exePath)
	ConfigureBackgroundProcess(cmd)
	return cmd.Start()
}

// CleanupPreviousExe removes the .old backup ApplyUpdate leaves behind
// after a successful update. Safe to call unconditionally on every
// startup — a missing .old file is simply ignored, not an error.
// CleanupPreviousExe removes the .old backup ApplyUpdate leaves behind
// after a successful update. Safe to call unconditionally on every
// startup — a missing .old file is simply ignored, not an error.
//
// Retries with a short backoff instead of trying once: the new process
// launches almost immediately after ApplyUpdate renames the *running*
// old process's own exe to .old, so the old process is often still in
// the middle of shutting down (and therefore still holding that file
// open) at the exact moment the new process's very first startup code
// tries to delete it. A single attempt makes cleanup succeed or fail
// depending on exactly how fast the old process happens to exit — that's
// the "sometimes it's there, sometimes it isn't" behavior. Call this from
// a goroutine, not inline in startup — see main.go.
func CleanupPreviousExe(currentExe string) {
	old := currentExe + ".old"
	for i := 0; i < 10; i++ {
		err := os.Remove(old)
		if err == nil || os.IsNotExist(err) {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// ApplyUpdate swaps newExePath into place as currentExe and relaunches it.
// Windows allows renaming (but not overwriting) a running executable, so:
//  1. currentExe -> currentExe+".old" (frees up the original name)
//  2. newExePath -> currentExe (the update takes its place)
//  3. launch currentExe
//
// The caller should exit shortly after this returns successfully — the
// new process is now running independently.
func ApplyUpdate(currentExe, newExePath string) error {
	old := currentExe + ".old"
	_ = os.Remove(old) // clean up a leftover from a previous update, if any

	if err := os.Rename(currentExe, old); err != nil {
		return err
	}
	if err := os.Rename(newExePath, currentExe); err != nil {
		_ = os.Rename(old, currentExe) // best-effort rollback
		return err
	}
	return launchFn(currentExe)
}
