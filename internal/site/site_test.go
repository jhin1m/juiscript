package site

import "testing"

func TestSiteHomeDir(t *testing.T) {
	s := &Site{User: "site_example_com"}
	got := s.HomeDir("/home")
	if got != "/home/site_example_com" {
		t.Errorf("HomeDir = %q", got)
	}
}

func TestSiteDirLaravel(t *testing.T) {
	s := &Site{User: "site_example_com", Domain: "example.com", ProjectType: ProjectLaravel}
	got := s.SiteDir("/home")
	if got != "/home/site_example_com/example.com" {
		t.Errorf("SiteDir (laravel) = %q", got)
	}
}

func TestSiteDirWordPress(t *testing.T) {
	s := &Site{User: "site_example_com", Domain: "example.com", ProjectType: ProjectWordPress}
	got := s.SiteDir("/home")
	if got != "/home/site_example_com/public_html/example.com" {
		t.Errorf("SiteDir (wordpress) = %q", got)
	}
}

func TestPHPSocket(t *testing.T) {
	s := &Site{User: "site_example_com", PHPVersion: "8.3"}
	got := s.PHPSocket()
	want := "/run/php/php8.3-fpm-site_example_com.sock"
	if got != want {
		t.Errorf("PHPSocket = %q, want %q", got, want)
	}
}

func TestFPMPoolConfigPath(t *testing.T) {
	s := &Site{Domain: "example.com", PHPVersion: "8.3"}
	got := s.FPMPoolConfigPath()
	want := "/etc/php/8.3/fpm/pool.d/example.com.conf"
	if got != want {
		t.Errorf("FPMPoolConfigPath = %q, want %q", got, want)
	}
}

func TestNginxPaths(t *testing.T) {
	s := &Site{Domain: "example.com"}

	available := s.NginxConfigPath("/etc/nginx/sites-available")
	if available != "/etc/nginx/sites-available/example.com.conf" {
		t.Errorf("NginxConfigPath = %q", available)
	}

	enabled := s.NginxEnabledPath("/etc/nginx/sites-enabled")
	if enabled != "/etc/nginx/sites-enabled/example.com.conf" {
		t.Errorf("NginxEnabledPath = %q", enabled)
	}
}
