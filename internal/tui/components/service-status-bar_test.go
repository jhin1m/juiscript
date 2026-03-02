package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/service"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

func newTestBar() *ServiceStatusBar {
	return NewServiceStatusBar(theme.New())
}

func TestNewServiceStatusBar(t *testing.T) {
	bar := newTestBar()
	if bar == nil {
		t.Fatal("expected non-nil ServiceStatusBar")
	}
	if bar.theme == nil {
		t.Fatal("expected non-nil theme")
	}
}

func TestEmptyState(t *testing.T) {
	bar := newTestBar()
	view := bar.View()
	if !strings.Contains(view, "No services detected") {
		t.Errorf("expected 'No services detected', got %q", view)
	}
}

func TestErrorState(t *testing.T) {
	bar := newTestBar()
	bar.SetError("connection failed")
	view := bar.View()
	if !strings.Contains(view, "⚠") {
		t.Error("expected warning icon in error state")
	}
	if !strings.Contains(view, "connection failed") {
		t.Error("expected error message in view")
	}
}

func TestErrorClearedOnSetServices(t *testing.T) {
	bar := newTestBar()
	bar.SetError("some error")
	bar.SetServices([]service.Status{
		{Name: "nginx", Active: true, State: "active", MemoryMB: 45},
	})
	view := bar.View()
	if strings.Contains(view, "⚠") {
		t.Error("error should be cleared after SetServices")
	}
}

func TestActiveService(t *testing.T) {
	bar := newTestBar()
	bar.SetServices([]service.Status{
		{Name: "nginx", Active: true, State: "active", MemoryMB: 45.7},
	})
	view := bar.View()
	if !strings.Contains(view, "●") {
		t.Error("expected filled dot for active service")
	}
	if !strings.Contains(view, "nginx") {
		t.Error("expected service name")
	}
	if !strings.Contains(view, "45MB") {
		t.Error("expected memory in MB (rounded)")
	}
}

func TestInactiveService(t *testing.T) {
	bar := newTestBar()
	bar.SetServices([]service.Status{
		{Name: "nginx", Active: false, State: "inactive", MemoryMB: 0},
	})
	view := bar.View()
	if !strings.Contains(view, "○") {
		t.Error("expected hollow dot for inactive service")
	}
	// Memory should NOT be shown for inactive services
	if strings.Contains(view, "MB") {
		t.Error("should not show memory for inactive services")
	}
}

func TestFailedService(t *testing.T) {
	bar := newTestBar()
	bar.SetServices([]service.Status{
		{Name: "mariadb", Active: false, State: "failed", MemoryMB: 0},
	})
	view := bar.View()
	if !strings.Contains(view, "●") {
		t.Error("expected filled dot for failed service")
	}
}

func TestFormatServiceName(t *testing.T) {
	bar := newTestBar()
	tests := []struct {
		input    string
		expected string
	}{
		{"php8.3-fpm", "php8.3"},
		{"php7.4-fpm", "php7.4"},
		{"redis-server", "redis"},
		{"nginx", "nginx"},
		{"mariadb", "mariadb"},
	}
	for _, tt := range tests {
		got := bar.formatServiceName(tt.input)
		if got != tt.expected {
			t.Errorf("formatServiceName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMultipleServices(t *testing.T) {
	bar := newTestBar()
	bar.SetServices([]service.Status{
		{Name: "nginx", Active: true, State: "active", MemoryMB: 45},
		{Name: "php8.3-fpm", Active: true, State: "active", MemoryMB: 32},
		{Name: "mariadb", Active: true, State: "active", MemoryMB: 120},
		{Name: "redis-server", Active: false, State: "inactive"},
	})
	view := bar.View()

	// Check separator exists
	if !strings.Contains(view, "│") {
		t.Error("expected pipe separator between services")
	}

	// Check PHP name shortened
	if strings.Contains(view, "php8.3-fpm") {
		t.Error("PHP name should be shortened to php8.3")
	}
	if !strings.Contains(view, "php8.3") {
		t.Error("expected shortened PHP name")
	}

	// Check redis name shortened
	if strings.Contains(view, "redis-server") {
		t.Error("redis name should be shortened to redis")
	}
}

func TestTruncation(t *testing.T) {
	bar := newTestBar()
	bar.SetWidth(40) // narrow width
	bar.SetServices([]service.Status{
		{Name: "nginx", Active: true, State: "active", MemoryMB: 45},
		{Name: "php8.3-fpm", Active: true, State: "active", MemoryMB: 32},
		{Name: "mariadb", Active: true, State: "active", MemoryMB: 120},
		{Name: "redis-server", Active: true, State: "active", MemoryMB: 8},
	})
	view := bar.View()

	// With 40 char width, should truncate some services
	renderedWidth := lipgloss.Width(view)
	if renderedWidth > 40 {
		t.Errorf("view width %d exceeds max width 40", renderedWidth)
	}

	if !strings.Contains(view, "+") || !strings.Contains(view, "more") {
		t.Error("expected '+N more' truncation indicator")
	}
}

func TestNoTruncationWhenWideEnough(t *testing.T) {
	bar := newTestBar()
	bar.SetWidth(200) // very wide
	bar.SetServices([]service.Status{
		{Name: "nginx", Active: true, State: "active", MemoryMB: 45},
		{Name: "mariadb", Active: true, State: "active", MemoryMB: 120},
	})
	view := bar.View()

	if strings.Contains(view, "more") {
		t.Error("should not truncate when width is sufficient")
	}
}

func TestActiveServiceWithZeroMemory(t *testing.T) {
	bar := newTestBar()
	bar.SetServices([]service.Status{
		{Name: "nginx", Active: true, State: "active", MemoryMB: 0},
	})
	view := bar.View()

	// Should NOT show "0MB"
	if strings.Contains(view, "0MB") {
		t.Error("should not show 0MB for active services with no memory data")
	}
}

func TestTruncationExtreme(t *testing.T) {
	bar := newTestBar()
	bar.SetWidth(5) // extremely narrow — can't fit even one segment
	bar.SetServices([]service.Status{
		{Name: "nginx", Active: true, State: "active", MemoryMB: 45},
		{Name: "mariadb", Active: true, State: "active", MemoryMB: 120},
	})
	view := bar.View()

	// Should show "+N more" fallback
	if !strings.Contains(view, "+2 more") {
		t.Errorf("expected '+2 more' fallback, got %q", view)
	}
}

func TestTruncationZeroWidth(t *testing.T) {
	bar := newTestBar()
	bar.SetWidth(2) // width <= padding, should return empty
	bar.SetServices([]service.Status{
		{Name: "nginx", Active: true, State: "active", MemoryMB: 45},
	})
	view := bar.View()
	if view != "" {
		t.Errorf("expected empty string for zero effective width, got %q", view)
	}
}

func TestSetWidth(t *testing.T) {
	bar := newTestBar()
	bar.SetWidth(80)
	if bar.width != 80 {
		t.Errorf("expected width 80, got %d", bar.width)
	}
}
