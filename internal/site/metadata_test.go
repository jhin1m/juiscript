package site

import (
	"testing"
	"time"
)

func TestSaveAndLoadMetadata(t *testing.T) {
	dir := t.TempDir()

	site := &Site{
		Domain:      "example.com",
		User:        "site_example_com",
		ProjectType: ProjectLaravel,
		PHPVersion:  "8.3",
		WebRoot:     "/home/site_example_com/example.com/public",
		Enabled:     true,
		CreatedAt:   time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
	}

	// Save
	if err := SaveMetadata(dir, site); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Load
	loaded, err := LoadMetadata(dir, "example.com")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded.Domain != site.Domain {
		t.Errorf("domain mismatch: got %s", loaded.Domain)
	}
	if loaded.User != site.User {
		t.Errorf("user mismatch: got %s", loaded.User)
	}
	if loaded.ProjectType != site.ProjectType {
		t.Errorf("project type mismatch: got %s", loaded.ProjectType)
	}
	if loaded.PHPVersion != site.PHPVersion {
		t.Errorf("php version mismatch: got %s", loaded.PHPVersion)
	}
	if loaded.Enabled != site.Enabled {
		t.Errorf("enabled mismatch: got %v", loaded.Enabled)
	}
}

func TestLoadAllMetadata(t *testing.T) {
	dir := t.TempDir()

	// Save two sites
	sites := []*Site{
		{Domain: "beta.com", User: "site_beta_com", ProjectType: ProjectWordPress, PHPVersion: "8.1", Enabled: true, CreatedAt: time.Now()},
		{Domain: "alpha.com", User: "site_alpha_com", ProjectType: ProjectLaravel, PHPVersion: "8.3", Enabled: true, CreatedAt: time.Now()},
	}

	for _, s := range sites {
		if err := SaveMetadata(dir, s); err != nil {
			t.Fatalf("save %s failed: %v", s.Domain, err)
		}
	}

	// Load all - should be sorted by domain
	loaded, err := LoadAllMetadata(dir)
	if err != nil {
		t.Fatalf("load all failed: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 sites, got %d", len(loaded))
	}

	// Should be sorted: alpha before beta
	if loaded[0].Domain != "alpha.com" {
		t.Errorf("expected first site alpha.com, got %s", loaded[0].Domain)
	}
	if loaded[1].Domain != "beta.com" {
		t.Errorf("expected second site beta.com, got %s", loaded[1].Domain)
	}
}

func TestLoadAllMetadataEmptyDir(t *testing.T) {
	// Non-existent dir should return nil, nil (no error)
	sites, err := LoadAllMetadata("/tmp/nonexistent-juiscript-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sites != nil {
		t.Errorf("expected nil sites, got %d", len(sites))
	}
}

func TestDeleteMetadata(t *testing.T) {
	dir := t.TempDir()

	site := &Site{
		Domain:    "delete-me.com",
		User:      "site_delete_me_com",
		Enabled:   true,
		CreatedAt: time.Now(),
	}
	SaveMetadata(dir, site)

	if err := DeleteMetadata(dir, "delete-me.com"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Should not be loadable anymore
	_, err := LoadMetadata(dir, "delete-me.com")
	if err == nil {
		t.Error("expected error loading deleted site")
	}
}
