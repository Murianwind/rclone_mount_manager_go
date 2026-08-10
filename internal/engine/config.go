package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Remote mirrors a parsed rclone.conf remote entry.
type Remote struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Config mirrors the structure stored in mounts.json.
type Config struct {
	Remotes        []Remote `json:"remotes"`
	Mounts         []Mount  `json:"mounts"`
	RclonePath     string   `json:"rclone_path"`
	AutoMount      bool     `json:"auto_mount"`
	StartMinimized bool     `json:"start_minimized"`
	WindowWidth    float32  `json:"window_width"`
	WindowHeight   float32  `json:"window_height"`
}

// DefaultConfig mirrors the Python fallback default returned by load_config().
func DefaultConfig() Config {
	return Config{
		Remotes:        []Remote{},
		Mounts:         []Mount{},
		RclonePath:     "",
		AutoMount:      false,
		StartMinimized: false,
	}
}

// Store handles reading/writing mounts.json in a given directory, including
// the backup/corruption-recovery behavior of load_config/save_config.
type Store struct {
	// Dir is the directory containing mounts.json / mounts.json.bak.
	Dir string
	// Log receives diagnostic messages (level: "INFO"|"WARN"|"ERROR").
	// Optional — a nil Log is treated as a no-op, matching write_log's
	// "never let logging failures break the caller" behavior.
	Log func(level, message string)
}

func (s Store) configPath() string { return filepath.Join(s.Dir, "mounts.json") }
func (s Store) backupPath() string { return filepath.Join(s.Dir, "mounts.json.bak") }

func (s Store) log(level, msg string) {
	if s.Log != nil {
		s.Log(level, msg)
	}
}

// Load reads mounts.json, recovering from mounts.json.bak if the primary
// file is corrupt, and preserving (never deleting) a corrupted file that
// can't be recovered. Mirrors load_config().
//
// Recovery order:
//  1. mounts.json parses cleanly -> return it
//  2. parse fails -> try mounts.json.bak; on success, re-save it as the
//     new mounts.json and return it
//  3. backup missing/also corrupt -> rename the corrupted original to
//     mounts.json.corrupted-<timestamp> (kept for manual recovery) and
//     return DefaultConfig()
func (s Store) Load() (Config, error) {
	path := s.configPath()
	if _, err := os.Stat(path); err != nil {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err == nil {
		var cfg Config
		if err := json.Unmarshal(data, &cfg); err == nil {
			if cfg.Mounts == nil {
				cfg.Mounts = []Mount{}
			}
			return cfg, nil
		}
	}
	s.log("ERROR", fmt.Sprintf("[설정] mounts.json 파싱 실패: %v", err))

	// 1차 복구: mounts.json.bak
	if bdata, berr := os.ReadFile(s.backupPath()); berr == nil {
		var cfg Config
		if uerr := json.Unmarshal(bdata, &cfg); uerr == nil {
			if cfg.Mounts == nil {
				cfg.Mounts = []Mount{}
			}
			s.log("INFO", "[설정] mounts.json.bak 으로 자동 복구 완료")
			_ = s.Save(cfg) // best-effort; failure here shouldn't block recovery
			return cfg, nil
		}
		s.log("ERROR", "[설정] mounts.json.bak 도 손상됨")
	}

	// 2차: 손상된 원본을 보존 (삭제하지 않고 타임스탬프 붙여 이름 변경)
	stamp := time.Now().Format("20060102_150405")
	corrupted := filepath.Join(s.Dir, fmt.Sprintf("mounts.json.corrupted-%s", stamp))
	if rerr := os.Rename(path, corrupted); rerr != nil {
		s.log("ERROR", fmt.Sprintf("[설정] 손상 파일 보존 실패: %v", rerr))
	} else {
		s.log("WARN", fmt.Sprintf("[설정] 손상된 파일 보존: %s", corrupted))
	}

	return DefaultConfig(), nil
}

// Save writes cfg to mounts.json atomically (write to a temp file, then
// rename over the target), backing up the previous valid file to
// mounts.json.bak first. Mirrors save_config().
func (s Store) Save(cfg Config) error {
	path := s.configPath()

	// 기존 파일이 유효한 JSON이면 백업 (손상된 파일로 백업을 덮어쓰지 않음)
	if existing, err := os.ReadFile(path); err == nil {
		if json.Valid(existing) {
			_ = os.WriteFile(s.backupPath(), existing, 0o644)
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	tmp := filepath.Join(s.Dir, "mounts.json.tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
