package site

import (
	"fmt"
	"regexp"
	"strings"
)

// domainRegex validates domain format.
// Allows: example.com, sub.example.com, my-site.example.co.uk
var domainRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*\.[a-z]{2,}$`)

// maxUsernameLen is the Linux username length limit.
const maxUsernameLen = 32

// ValidateDomain checks if a domain name is valid.
func ValidateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// Normalize to lowercase
	domain = strings.ToLower(domain)

	if len(domain) > 253 {
		return fmt.Errorf("domain too long (max 253 chars)")
	}

	if !domainRegex.MatchString(domain) {
		return fmt.Errorf("invalid domain format: %s", domain)
	}

	return nil
}

// DeriveUsername creates a Linux username from a domain.
// Example: "example.com" -> "site_example_com"
// Ensures the result is safe for useradd (alphanumeric + underscore).
func DeriveUsername(domain string) string {
	// Replace dots and hyphens with underscores
	name := strings.NewReplacer(
		".", "_",
		"-", "_",
	).Replace(strings.ToLower(domain))

	name = "site_" + name

	// Truncate to max length
	if len(name) > maxUsernameLen {
		name = name[:maxUsernameLen]
	}

	return name
}

// ValidateProjectType checks if the project type is supported.
func ValidateProjectType(pt ProjectType) error {
	switch pt {
	case ProjectLaravel, ProjectWordPress:
		return nil
	default:
		return fmt.Errorf("unsupported project type: %s (use 'laravel' or 'wordpress')", pt)
	}
}

// ValidatePHPVersion checks if a PHP version string is valid format.
func ValidatePHPVersion(version string) error {
	if version == "" {
		return fmt.Errorf("PHP version cannot be empty")
	}

	// Simple format check: X.Y (e.g., "8.3", "8.1")
	phpVersionRegex := regexp.MustCompile(`^[0-9]+\.[0-9]+$`)
	if !phpVersionRegex.MatchString(version) {
		return fmt.Errorf("invalid PHP version format: %s (expected X.Y, e.g. 8.3)", version)
	}

	return nil
}
