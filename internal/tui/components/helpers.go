package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// statusIndicator returns a colored dot and style based on service state.
// Shared by ServicePanel and ServiceStatusBar.
func statusIndicator(t *theme.Theme, state string) (string, lipgloss.Style) {
	switch state {
	case "active":
		return "●", t.OkText
	case "failed":
		return "●", t.ErrorText
	default:
		return "○", t.Subtitle
	}
}
