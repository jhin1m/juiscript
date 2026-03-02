package template

import (
	"strings"
	"testing"
)

func TestNewEngine(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	templates := engine.Available()
	if len(templates) == 0 {
		t.Fatal("expected at least one template")
	}
}

func TestRenderNginxVhost(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	data := struct {
		Domain      string
		Aliases     string
		WebRoot     string
		User        string
		ProjectType string
		PHPSocket   string
	}{
		Domain:      "example.com",
		Aliases:     "www.example.com",
		WebRoot:     "/home/site_example_com/example.com/public",
		User:        "site_example_com",
		ProjectType: "laravel",
		PHPSocket:   "/run/php/php8.3-fpm-site_example_com.sock",
	}

	result, err := engine.Render("nginx-vhost.conf.tmpl", data)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	// Verify key parts are present
	checks := []string{
		"server_name example.com",
		"www.example.com",
		"/home/site_example_com",
		"try_files $uri $uri/ /index.php?$query_string", // Laravel-specific
		"fastcgi_pass unix:/run/php/php8.3-fpm",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("expected output to contain %q", check)
		}
	}
}

func TestRenderPHPFPMPool(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	data := struct {
		PoolName   string
		User       string
		SocketPath string
		MaxChildren int
		StartServers int
		MinSpare     int
		MaxSpare     int
	}{
		PoolName:     "site_example_com",
		User:         "site_example_com",
		SocketPath:   "/run/php/php8.3-fpm-site_example_com.sock",
		MaxChildren:  5,
		StartServers: 2,
		MinSpare:     1,
		MaxSpare:     3,
	}

	result, err := engine.Render("php-fpm-pool.conf.tmpl", data)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if !strings.Contains(result, "[site_example_com]") {
		t.Error("expected pool name section")
	}
	if !strings.Contains(result, "user = site_example_com") {
		t.Error("expected user directive")
	}
	if !strings.Contains(result, "open_basedir") {
		t.Error("expected open_basedir security restriction")
	}
}
