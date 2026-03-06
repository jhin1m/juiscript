package components

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// confirmPanelStyle is the cached panel style for the confirm dialog.
var confirmPanelStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(theme.Warning).
	Padding(1, 2)

// ConfirmYesMsg signals user confirmed the destructive action.
type ConfirmYesMsg struct{}

// ConfirmNoMsg signals user cancelled.
type ConfirmNoMsg struct{}

// ConfirmModel is an inline confirmation dialog for destructive actions.
// Default selection is No (safe default).
type ConfirmModel struct {
	theme    *theme.Theme
	message  string
	selected bool // false=No (safe default), true=Yes
	active   bool
}

// NewConfirm creates a new confirmation dialog.
func NewConfirm(t *theme.Theme) *ConfirmModel {
	return &ConfirmModel{theme: t}
}

// Show activates the dialog with a warning message. Resets selection to No.
func (c *ConfirmModel) Show(message string) {
	c.message = message
	c.selected = false // safe default
	c.active = true
}

// Active returns whether the dialog is visible.
func (c *ConfirmModel) Active() bool {
	return c.active
}

// Update handles key input for the confirmation dialog.
func (c *ConfirmModel) Update(msg tea.Msg) (*ConfirmModel, tea.Cmd) {
	if !c.active {
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y":
			c.active = false
			return c, func() tea.Msg { return ConfirmYesMsg{} }
		case "n", "esc":
			c.active = false
			return c, func() tea.Msg { return ConfirmNoMsg{} }
		case "tab", "left", "right", "h", "l":
			c.selected = !c.selected
		case "enter":
			c.active = false
			if c.selected {
				return c, func() tea.Msg { return ConfirmYesMsg{} }
			}
			return c, func() tea.Msg { return ConfirmNoMsg{} }
		}
	}
	return c, nil
}

// View renders the warning dialog.
func (c *ConfirmModel) View() string {
	if !c.active {
		return ""
	}

	// Warning header
	header := c.theme.WarnText.Bold(true).Render("  WARNING")

	// Message
	body := fmt.Sprintf("\n  %s\n", c.message)

	// Buttons: No is default (safe), Yes is dangerous
	var noBtn, yesBtn string
	if c.selected {
		noBtn = c.theme.Subtitle.Render("[ No ]")
		yesBtn = c.theme.ErrorText.Bold(true).Render("[ Yes ]")
	} else {
		noBtn = c.theme.OkText.Bold(true).Render("[ No ]")
		yesBtn = c.theme.Subtitle.Render("[ Yes ]")
	}
	buttons := fmt.Sprintf("\n  %s   %s", noBtn, yesBtn)

	help := c.theme.HelpDesc.Render("\n\n  y:yes  n/esc:no  tab:toggle  enter:confirm")

	// Wrap in warning-styled panel (style cached at package level)
	content := header + body + buttons + help
	panel := confirmPanelStyle.Render(content)

	return panel
}
