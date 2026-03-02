package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/service"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// ServicePanel renders a compact service status overview for the dashboard.
// Shows green/red dots next to each service name.
type ServicePanel struct {
	theme    *theme.Theme
	services []service.Status
}

// NewServicePanel creates a reusable service status panel.
func NewServicePanel(t *theme.Theme) *ServicePanel {
	return &ServicePanel{theme: t}
}

// SetServices updates the displayed service statuses.
func (p *ServicePanel) SetServices(services []service.Status) {
	p.services = services
}

// View renders the compact service panel.
func (p *ServicePanel) View() string {
	if len(p.services) == 0 {
		return p.theme.Subtitle.Render("  No services detected")
	}

	title := p.theme.HelpKey.Render("Service Status")

	var lines []string
	for _, svc := range p.services {
		dot, style := statusIndicator(p.theme, svc.State)
		name := string(svc.Name)
		line := fmt.Sprintf("  %s %s", style.Render(dot), p.theme.Subtitle.Render(name))
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	return lipgloss.JoinVertical(lipgloss.Left, title, content)
}

