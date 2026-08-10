package engine

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	sectionHeaderRe = regexp.MustCompile(`^\[(.+)\]$`)
	typeLineRe      = regexp.MustCompile(`^type\s*=\s*(.+)$`)
)

// ParseRcloneConf reads an rclone.conf (INI-shaped: "[name]" sections,
// "type = xxx" lines) and returns each remote's name and type. Mirrors
// parse_rclone_conf() — including its "never error, just return what
// could be parsed" behavior, since a partially-malformed conf file
// shouldn't block importing the remotes that DID parse fine.
func ParseRcloneConf(confPath string) []Remote {
	f, err := os.Open(confPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var remotes []Remote
	var current *Remote

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if m := sectionHeaderRe.FindStringSubmatch(line); m != nil {
			if current != nil {
				remotes = append(remotes, *current)
			}
			current = &Remote{Name: m[1]}
			continue
		}
		if current == nil {
			continue // 아직 어떤 섹션에도 들어가기 전의 줄(주석 등)은 건너뜀
		}
		if m := typeLineRe.FindStringSubmatch(line); m != nil {
			current.Type = strings.TrimSpace(m[1])
		}
	}
	if current != nil {
		remotes = append(remotes, *current)
	}
	return remotes
}

// FindDefaultRcloneConf looks for rclone.conf in the locations rclone
// itself checks by default, in priority order, mirroring
// find_default_rclone_conf(). appDir is the running exe's directory
// (the last, lowest-priority fallback).
func FindDefaultRcloneConf(appDir string) (string, bool) {
	candidates := []string{
		filepath.Join(os.Getenv("APPDATA"), "rclone", "rclone.conf"),
		filepath.Join(userHomeDir(), ".config", "rclone", "rclone.conf"),
		filepath.Join(appDir, "rclone.conf"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
