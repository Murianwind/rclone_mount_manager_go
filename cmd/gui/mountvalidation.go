package main

import "github.com/Murianwind/rclone-manager-go/internal/engine"

// validateMount checks a mount for save-blocking conflicts against the
// rest of cfg.Mounts (ignoring the mount itself when editing, matched by
// ID). Returns "" if there's no conflict, else a user-facing message.
// Pulled out as a pure function for testing — see mountvalidation_test.go.
func validateMount(m engine.Mount, existingMounts []engine.Mount) string {
	drive := m.Drive
	for _, other := range existingMounts {
		if other.ID == m.ID {
			continue
		}
		if drive != "" && other.Drive == drive {
			return "이미 사용 중인 드라이브 문자입니다."
		}
		if other.Remote == m.Remote && other.RemotePath == m.RemotePath {
			return "동일한 리모트/경로가 이미 등록되어 있습니다."
		}
	}
	return ""
}
