package site

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// SaveMetadata writes site metadata to a TOML file.
func SaveMetadata(sitesDir string, s *Site) error {
	path := s.MetadataPath(sitesDir)

	if err := os.MkdirAll(sitesDir, 0750); err != nil {
		return fmt.Errorf("create sites dir: %w", err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		return fmt.Errorf("create metadata file: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(s); err != nil {
		return fmt.Errorf("encode site metadata: %w", err)
	}

	return nil
}

// LoadMetadata reads a single site from its TOML file.
func LoadMetadata(sitesDir, domain string) (*Site, error) {
	path := filepath.Join(sitesDir, domain+".toml")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read site metadata: %w", err)
	}

	var site Site
	if err := toml.Unmarshal(data, &site); err != nil {
		return nil, fmt.Errorf("parse site metadata: %w", err)
	}

	return &site, nil
}

// LoadAllMetadata reads all site TOML files from the sites directory.
// Returns sites sorted by domain name.
func LoadAllMetadata(sitesDir string) ([]*Site, error) {
	entries, err := os.ReadDir(sitesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no sites yet
		}
		return nil, fmt.Errorf("read sites dir: %w", err)
	}

	var sites []*Site
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		domain := strings.TrimSuffix(entry.Name(), ".toml")
		site, err := LoadMetadata(sitesDir, domain)
		if err != nil {
			// Log warning but continue loading other sites
			continue
		}
		sites = append(sites, site)
	}

	// Sort by domain for consistent ordering
	sort.Slice(sites, func(i, j int) bool {
		return sites[i].Domain < sites[j].Domain
	})

	return sites, nil
}

// DeleteMetadata removes a site's TOML metadata file.
func DeleteMetadata(sitesDir, domain string) error {
	path := filepath.Join(sitesDir, domain+".toml")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete metadata: %w", err)
	}
	return nil
}
