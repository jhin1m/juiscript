package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.General.SitesRoot != "/home" {
		t.Errorf("expected SitesRoot /home, got %s", cfg.General.SitesRoot)
	}
	if cfg.PHP.DefaultVersion != "8.3" {
		t.Errorf("expected PHP 8.3, got %s", cfg.PHP.DefaultVersion)
	}
	if cfg.Backup.RetentionDays != 30 {
		t.Errorf("expected 30 retention days, got %d", cfg.Backup.RetentionDays)
	}
}

func TestLoadMissing(t *testing.T) {
	// Load from non-existent path should return defaults
	cfg, err := Load("/tmp/nonexistent-juiscript-test.toml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.General.SitesRoot != "/home" {
		t.Errorf("expected default SitesRoot, got %s", cfg.General.SitesRoot)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Save custom config
	cfg := Default()
	cfg.General.SitesRoot = "/custom/sites"
	cfg.PHP.DefaultVersion = "8.1"

	if err := cfg.Save(path); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Verify file permissions (0640)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0640 {
		t.Errorf("expected 0640 permissions, got %o", perm)
	}

	// Load it back
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.General.SitesRoot != "/custom/sites" {
		t.Errorf("expected /custom/sites, got %s", loaded.General.SitesRoot)
	}
	if loaded.PHP.DefaultVersion != "8.1" {
		t.Errorf("expected 8.1, got %s", loaded.PHP.DefaultVersion)
	}
}

func TestLoadInvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")

	// Write invalid TOML
	os.WriteFile(path, []byte("{{invalid"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid TOML")
	}
}
