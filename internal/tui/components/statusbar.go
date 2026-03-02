package components

import (
	"strings"

	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// StatusBar renders keyboard shortcuts at the bottom.
type StatusBar struct {
	theme *theme.Theme
	width int
}

func NewStatusBar(t *theme.Theme) *StatusBar {
	return &StatusBar{theme: t}
}

func (s *StatusBar) SetWidth(w int) {
	s.width = w
}

// KeyBinding represents a keyboard shortcut hint.
type KeyBinding struct {
	Key  string
	Desc string
}

// View renders the status bar with the given key bindings.
func (s *StatusBar) View(bindings []KeyBinding) string {
	var parts []string
	for _, b := range bindings {
		key := s.theme.HelpKey.Render(b.Key)
		desc := s.theme.HelpDesc.Render(b.Desc)
		parts = append(parts, key+" "+desc)
	}

	content := strings.Join(parts, "  ")
	return s.theme.StatusBar.Width(s.width).Render(content)
}
