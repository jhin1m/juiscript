package screens

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/nginx"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// NginxScreen displays all vhosts with enable/disable/delete actions.
type NginxScreen struct {
	theme  *theme.Theme
	vhosts []nginx.VhostInfo
	cursor int
	width  int
	height int
	err    error
}

// NewNginxScreen creates the Nginx vhost management screen.
func NewNginxScreen(t *theme.Theme) *NginxScreen {
	return &NginxScreen{theme: t}
}

// SetVhosts updates the vhost list data.
func (n *NginxScreen) SetVhosts(vhosts []nginx.VhostInfo) {
	n.vhosts = vhosts
	n.err = nil
}

// SetError sets an error to display.
func (n *NginxScreen) SetError(err error) {
	n.err = err
}

func (n *NginxScreen) Init() tea.Cmd { return nil }

func (n *NginxScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		n.width = msg.Width
		n.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if n.cursor > 0 {
				n.cursor--
			}
		case "down", "j":
			if n.cursor < len(n.vhosts)-1 {
				n.cursor++
			}
		case "e":
			if len(n.vhosts) > 0 {
				v := n.vhosts[n.cursor]
				return n, func() tea.Msg {
					return ToggleVhostMsg{Domain: v.Domain, CurrentlyEnabled: v.Enabled}
				}
			}
		case "d":
			if len(n.vhosts) > 0 {
				return n, func() tea.Msg {
					return DeleteVhostMsg{Domain: n.vhosts[n.cursor].Domain}
				}
			}
		case "t":
			return n, func() tea.Msg {
				return TestNginxMsg{}
			}
		case "esc", "q":
			return n, func() tea.Msg { return GoBackMsg{} }
		}
	}

	return n, nil
}

func (n *NginxScreen) View() string {
	title := n.theme.Title.Render("Nginx Vhosts")

	if n.err != nil {
		errMsg := n.theme.ErrorText.Render(fmt.Sprintf("Error: %v", n.err))
		return lipgloss.JoinVertical(lipgloss.Left, title, "", errMsg)
	}

	if len(n.vhosts) == 0 {
		empty := n.theme.Subtitle.Render("  No vhosts configured. Create a site first.")
		return lipgloss.JoinVertical(lipgloss.Left, title, "", empty)
	}

	// Table header
	header := fmt.Sprintf("  %-35s %-10s %s", "DOMAIN", "STATUS", "PATH")
	headerStyle := n.theme.HelpKey.Render(header)

	// Table rows
	var rows string
	for i, v := range n.vhosts {
		cursor := "  "
		style := n.theme.Inactive
		if i == n.cursor {
			cursor = "> "
			style = n.theme.Active
		}

		status := "enabled"
		statusStyle := n.theme.OkText
		if !v.Enabled {
			status = "disabled"
			statusStyle = n.theme.ErrorText
		}

		row := fmt.Sprintf("%s%-35s %s  %s",
			cursor,
			style.Render(v.Domain),
			statusStyle.Render(fmt.Sprintf("%-10s", status)),
			n.theme.Subtitle.Render(v.Path),
		)
		rows += row + "\n"
	}

	help := n.theme.HelpDesc.Render("  e:enable/disable  d:delete  t:test config  esc:back")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, "", headerStyle, rows, help)
}

func (n *NginxScreen) ScreenTitle() string { return "Nginx" }

// Messages for nginx screen actions
type ToggleVhostMsg struct {
	Domain           string
	CurrentlyEnabled bool
}

type DeleteVhostMsg struct {
	Domain string
}

type TestNginxMsg struct{}
