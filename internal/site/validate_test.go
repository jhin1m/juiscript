package site

import "testing"

func TestValidateDomain(t *testing.T) {
	valid := []string{
		"example.com",
		"sub.example.com",
		"my-site.example.co.uk",
		"a.io",
		"test123.dev",
	}

	for _, d := range valid {
		if err := ValidateDomain(d); err != nil {
			t.Errorf("expected %q to be valid, got: %v", d, err)
		}
	}

	invalid := []string{
		"",
		"example",           // no TLD
		"-example.com",      // starts with hyphen
		"example-.com",      // ends with hyphen
		"exam ple.com",      // space
		"a.b",               // TLD too short
		".example.com",      // starts with dot
	}

	for _, d := range invalid {
		if err := ValidateDomain(d); err == nil {
			t.Errorf("expected %q to be invalid", d)
		}
	}
}

func TestDeriveUsername(t *testing.T) {
	tests := []struct {
		domain   string
		expected string
	}{
		{"example.com", "site_example_com"},
		{"my-site.io", "site_my_site_io"},
		{"sub.domain.co.uk", "site_sub_domain_co_uk"},
	}

	for _, tt := range tests {
		got := DeriveUsername(tt.domain)
		if got != tt.expected {
			t.Errorf("DeriveUsername(%q) = %q, want %q", tt.domain, got, tt.expected)
		}
	}
}

func TestDeriveUsernameTruncation(t *testing.T) {
	// Very long domain should be truncated to 32 chars
	long := "very-long-subdomain.very-long-domain.com"
	result := DeriveUsername(long)
	if len(result) > maxUsernameLen {
		t.Errorf("username too long: %d chars (max %d)", len(result), maxUsernameLen)
	}
}

func TestValidateProjectType(t *testing.T) {
	if err := ValidateProjectType(ProjectLaravel); err != nil {
		t.Errorf("laravel should be valid: %v", err)
	}
	if err := ValidateProjectType(ProjectWordPress); err != nil {
		t.Errorf("wordpress should be valid: %v", err)
	}
	if err := ValidateProjectType("flask"); err == nil {
		t.Error("flask should be invalid")
	}
}

func TestValidatePHPVersion(t *testing.T) {
	valid := []string{"8.3", "8.1", "7.4"}
	for _, v := range valid {
		if err := ValidatePHPVersion(v); err != nil {
			t.Errorf("%q should be valid: %v", v, err)
		}
	}

	invalid := []string{"", "8", "8.3.1", "abc"}
	for _, v := range invalid {
		if err := ValidatePHPVersion(v); err == nil {
			t.Errorf("%q should be invalid", v)
		}
	}
}
