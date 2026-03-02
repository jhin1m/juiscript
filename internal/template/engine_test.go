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

func TestRenderNginxLaravelVhost(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	data := struct {
		Domain      string
		WebRoot     string
		PHPSocket   string
		AccessLog   string
		ErrorLog    string
		MaxBodySize string
		ExtraConfig string
	}{
		Domain:      "example.com",
		WebRoot:     "/home/site_example_com/example.com/public",
		PHPSocket:   "/run/php/php8.3-fpm-site_example_com.sock",
		AccessLog:   "/home/site_example_com/logs/nginx-access.log",
		ErrorLog:    "/home/site_example_com/logs/nginx-error.log",
		MaxBodySize: "64m",
	}

	result, err := engine.Render("nginx-laravel.conf.tmpl", data)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	checks := []string{
		"server_name example.com",
		"/home/site_example_com",
		"try_files $uri $uri/ /index.php?$query_string",
		"fastcgi_pass unix:/run/php/php8.3-fpm",
		"client_max_body_size 64m",
		"X-Frame-Options",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("expected output to contain %q", check)
		}
	}
}

func TestRenderNginxWordPressVhost(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	data := struct {
		Domain      string
		WebRoot     string
		PHPSocket   string
		AccessLog   string
		ErrorLog    string
		MaxBodySize string
		ExtraConfig string
	}{
		Domain:      "blog.example.com",
		WebRoot:     "/home/site_blog/public_html/blog.example.com",
		PHPSocket:   "/run/php/php8.3-fpm-site_blog.sock",
		AccessLog:   "/home/site_blog/logs/nginx-access.log",
		ErrorLog:    "/home/site_blog/logs/nginx-error.log",
		MaxBodySize: "128m",
	}

	result, err := engine.Render("nginx-wordpress.conf.tmpl", data)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	checks := []string{
		"server_name blog.example.com www.blog.example.com",
		"try_files $uri $uri/ /index.php?$args",
		"favicon.ico",
		"robots.txt",
		"client_max_body_size 128m",
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
		PoolName      string
		User          string
		SocketPath    string
		MaxChildren   int
		StartServers  int
		MinSpare      int
		MaxSpare      int
		MaxRequests   int
		MemoryLimit   string
		UploadMaxSize string
		Timezone      string
	}{
		PoolName:      "site_example_com",
		User:          "site_example_com",
		SocketPath:    "/run/php/php8.3-fpm-site_example_com.sock",
		MaxChildren:   5,
		StartServers:  2,
		MinSpare:      1,
		MaxSpare:      3,
		MaxRequests:   500,
		MemoryLimit:   "256M",
		UploadMaxSize: "64M",
		Timezone:      "UTC",
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
	if !strings.Contains(result, "memory_limit] = 256M") {
		t.Error("expected memory_limit setting")
	}
	if !strings.Contains(result, "security.limit_extensions = .php") {
		t.Error("expected security.limit_extensions")
	}
}
