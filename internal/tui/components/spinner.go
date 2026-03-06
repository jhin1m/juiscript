package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// SpinnerModel wraps bubbles spinner with theme styling.
// Screens embed this to show loading state during async operations.
type SpinnerModel struct {
	theme   *theme.Theme
	spinner spinner.Model
	message string
	active  bool
}

// NewSpinner creates a themed spinner with dot style.
func NewSpinner(t *theme.Theme) *SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = s.Style.Foreground(theme.Primary)
	return &SpinnerModel{
		theme:   t,
		spinner: s,
	}
}

// Start activates the spinner with a message and returns the tick cmd.
func (s *SpinnerModel) Start(message string) tea.Cmd {
	s.message = message
	s.active = true
	return s.spinner.Tick
}

// Stop deactivates the spinner.
func (s *SpinnerModel) Stop() {
	s.active = false
	s.message = ""
}

// Active returns whether the spinner is running.
func (s *SpinnerModel) Active() bool {
	return s.active
}

// Update forwards tick messages to the internal spinner when active.
func (s *SpinnerModel) Update(msg tea.Msg) (*SpinnerModel, tea.Cmd) {
	if !s.active {
		return s, nil
	}
	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}

// View returns the spinner animation + message, or empty string if inactive.
func (s *SpinnerModel) View() string {
	if !s.active {
		return ""
	}
	return fmt.Sprintf("  %s %s", s.spinner.View(), s.message)
}
