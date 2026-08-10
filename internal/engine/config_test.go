package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_NoFile_ReturnsDefault(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	cfg, err := s.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Mounts) != 0 {
		t.Errorf("expected empty Mounts, got %v", cfg.Mounts)
	}
}

func TestLoad_CorruptNoBackup_ReturnsDefaultAndPreservesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mounts.json")
	if err := os.WriteFile(path, []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := Store{Dir: dir}
	cfg, err := s.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Mounts) != 0 {
		t.Errorf("expected empty Mounts, got %v", cfg.Mounts)
	}
	// original mounts.json must be gone (renamed), never silently deleted
	if _, err := os.Stat(path); err == nil {
		t.Errorf("expected corrupt mounts.json to be renamed away")
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "mounts.json.corrupted-*"))
	if len(matches) != 1 {
		t.Errorf("expected exactly one preserved corrupted file, got %v", matches)
	}
}

func TestSave_NoExistingFile_WritesAtomically(t *testing.T) {
	dir := t.TempDir()
	s := Store{Dir: dir}
	if err := s.Save(Config{Mounts: []Mount{}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path := filepath.Join(dir, "mounts.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected mounts.json to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "mounts.json.tmp")); err == nil {
		t.Errorf("temp file should not remain after atomic rename")
	}
}

func TestSave_ExistingValidFile_CreatesBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mounts.json")
	if err := os.WriteFile(path, []byte(`{"mounts":[{"remote":"old"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s := Store{Dir: dir}
	if err := s.Save(Config{Mounts: []Mount{{Remote: "new"}}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	backup, err := os.ReadFile(filepath.Join(dir, "mounts.json.bak"))
	if err != nil {
		t.Fatalf("expected backup file: %v", err)
	}
	var backupCfg Config
	if err := json.Unmarshal(backup, &backupCfg); err != nil {
		t.Fatalf("backup is not valid JSON: %v", err)
	}
	if len(backupCfg.Mounts) != 1 || backupCfg.Mounts[0].Remote != "old" {
		t.Errorf("expected backup to contain the OLD config, got %+v", backupCfg)
	}

	current, _ := os.ReadFile(path)
	var currentCfg Config
	_ = json.Unmarshal(current, &currentCfg)
	if len(currentCfg.Mounts) != 1 || currentCfg.Mounts[0].Remote != "new" {
		t.Errorf("expected mounts.json to contain the NEW config, got %+v", currentCfg)
	}
}

func TestLoad_RecoversFromBackup(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mounts.json"), []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mounts.json.bak"),
		[]byte(`{"mounts":[{"remote":"backup-ok"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s := Store{Dir: dir}
	cfg, err := s.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Mounts) != 1 || cfg.Mounts[0].Remote != "backup-ok" {
		t.Errorf("expected recovery from backup, got %+v", cfg)
	}
}

func TestLoad_BackupAlsoCorrupt_PreservesOriginalReturnsDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mounts.json")
	if err := os.WriteFile(path, []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mounts.json.bak"), []byte("{also bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := Store{Dir: dir}
	cfg, err := s.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Mounts) != 0 {
		t.Errorf("expected default empty config, got %+v", cfg)
	}
	if _, err := os.Stat(path); err == nil {
		t.Errorf("expected corrupt original to be renamed away, not left in place")
	}
}

func TestLoad_MountsKeyAlreadyPresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mounts.json"),
		[]byte(`{"mounts":[{"remote":"x"}],"remotes":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s := Store{Dir: dir}
	cfg, err := s.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Mounts) != 1 {
		t.Errorf("expected 1 mount, got %d", len(cfg.Mounts))
	}
}
