package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/service"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// ServiceStatusBar renders a horizontal bar showing LEMP service health.
// Displayed between header and content on all TUI screens.
// Shows colored dots (green=active, red=failed, gray=inactive) with memory usage.
type ServiceStatusBar struct {
	theme    *theme.Theme
	services []service.Status
	width    int
	errMsg   string
}

// NewServiceStatusBar creates a service status bar with the given theme.
func NewServiceStatusBar(t *theme.Theme) *ServiceStatusBar {
	return &ServiceStatusBar{theme: t}
}

// SetServices updates the list of service statuses to display.
func (b *ServiceStatusBar) SetServices(services []service.Status) {
	b.services = services
	b.errMsg = "" // clear error when new data arrives
}

// SetWidth sets the available terminal width for truncation.
func (b *ServiceStatusBar) SetWidth(w int) {
	b.width = w
}

// SetError sets an error message to display instead of service statuses.
func (b *ServiceStatusBar) SetError(msg string) {
	b.errMsg = msg
}

// View renders the service status bar as a single horizontal line.
func (b *ServiceStatusBar) View() string {
	// Error state: show warning icon + message
	if b.errMsg != "" {
		return b.theme.WarnText.Render("  ⚠ " + b.errMsg)
	}

	// Empty state: no services detected
	if len(b.services) == 0 {
		return b.theme.Subtitle.Render("  ○ No services detected")
	}

	// Build a segment for each service
	segments := make([]string, 0, len(b.services))
	for _, svc := range b.services {
		segments = append(segments, b.renderSegment(svc))
	}

	separator := b.theme.Inactive.Render(" │ ")

	// Use truncation logic when width is set, otherwise join directly
	if b.width > 0 {
		return b.truncate(segments, separator)
	}
	return "  " + strings.Join(segments, separator)
}

// renderSegment builds a single service segment like "● nginx 45MB".
func (b *ServiceStatusBar) renderSegment(svc service.Status) string {
	dot, style := statusIndicator(b.theme, svc.State)
	name := b.formatServiceName(string(svc.Name))

	segment := style.Render(dot) + " " + b.theme.Subtitle.Render(name)

	// Show memory only for active services with non-zero usage
	if svc.Active && svc.MemoryMB > 0 {
		mem := fmt.Sprintf(" %dMB", int(svc.MemoryMB))
		segment += b.theme.Inactive.Render(mem)
	}

	return segment
}

// truncate fits segments into available width, adding "+N more" if needed.
func (b *ServiceStatusBar) truncate(segments []string, separator string) string {
	const padding = 2 // left padding "  "
	maxWidth := b.width - padding

	if maxWidth <= 0 {
		return ""
	}

	// Try fitting all segments first
	full := strings.Join(segments, separator)
	if lipgloss.Width(full) <= maxWidth {
		return "  " + full
	}

	// Progressively add segments until we exceed width
	for i := len(segments) - 1; i >= 1; i-- {
		remaining := len(segments) - i
		suffix := b.theme.Inactive.Render(fmt.Sprintf(" +%d more", remaining))

		partial := strings.Join(segments[:i], separator) + suffix
		if lipgloss.Width(partial) <= maxWidth {
			return "  " + partial
		}
	}

	// Can't fit even one segment + suffix: show just the suffix
	return "  " + b.theme.Inactive.Render(fmt.Sprintf("+%d more", len(segments)))
}

// formatServiceName shortens service names for compact display.
// "php8.3-fpm" → "php8.3", "redis-server" → "redis", others unchanged.
func (b *ServiceStatusBar) formatServiceName(name string) string {
	if strings.HasSuffix(name, "-fpm") {
		return strings.TrimSuffix(name, "-fpm")
	}
	if strings.HasSuffix(name, "-server") {
		return strings.TrimSuffix(name, "-server")
	}
	return name
}

