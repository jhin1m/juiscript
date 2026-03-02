package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// Header renders the app title bar with navigation context.
type Header struct {
	theme *theme.Theme
	width int
}

func NewHeader(t *theme.Theme) *Header {
	return &Header{theme: t}
}

func (h *Header) SetWidth(w int) {
	h.width = w
}

// View renders the header with current screen title.
func (h *Header) View(screenTitle string) string {
	title := h.theme.Header.Render(" juiscript ")
	screen := h.theme.Subtitle.Render(fmt.Sprintf(" > %s", screenTitle))

	bar := lipgloss.JoinHorizontal(lipgloss.Center, title, screen)

	return lipgloss.NewStyle().
		Width(h.width).
		Render(bar)
}
