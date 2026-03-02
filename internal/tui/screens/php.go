package screens

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/php"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// PHPScreen displays installed PHP versions and their FPM status.
type PHPScreen struct {
	theme    *theme.Theme
	versions []php.VersionInfo
	cursor   int
	width    int
	height   int
	err      error
}

// NewPHPScreen creates the PHP management screen.
func NewPHPScreen(t *theme.Theme) *PHPScreen {
	return &PHPScreen{theme: t}
}

// SetVersions updates the PHP version list.
func (p *PHPScreen) SetVersions(versions []php.VersionInfo) {
	p.versions = versions
	p.err = nil
}

// SetError sets an error to display.
func (p *PHPScreen) SetError(err error) {
	p.err = err
}

func (p *PHPScreen) Init() tea.Cmd { return nil }

func (p *PHPScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if p.cursor > 0 {
				p.cursor--
			}
		case "down", "j":
			if p.cursor < len(p.versions)-1 {
				p.cursor++
			}
		case "i":
			return p, func() tea.Msg {
				return InstallPHPMsg{}
			}
		case "r":
			if len(p.versions) > 0 {
				return p, func() tea.Msg {
					return RemovePHPMsg{Version: p.versions[p.cursor].Version}
				}
			}
		case "esc", "q":
			return p, func() tea.Msg { return GoBackMsg{} }
		}
	}

	return p, nil
}

func (p *PHPScreen) View() string {
	title := p.theme.Title.Render("PHP Versions")

	if p.err != nil {
		errMsg := p.theme.ErrorText.Render(fmt.Sprintf("Error: %v", p.err))
		return lipgloss.JoinVertical(lipgloss.Left, title, "", errMsg)
	}

	if len(p.versions) == 0 {
		empty := p.theme.Subtitle.Render("  No PHP versions installed.")
		help := p.theme.HelpDesc.Render("  i:install  esc:back")
		return lipgloss.JoinVertical(lipgloss.Left, title, "", empty, "", help)
	}

	// Table header
	header := fmt.Sprintf("  %-12s %-12s %s", "VERSION", "FPM STATUS", "BOOT")
	headerStyle := p.theme.HelpKey.Render(header)

	// Table rows
	var rows string
	for i, v := range p.versions {
		cursor := "  "
		style := p.theme.Inactive
		if i == p.cursor {
			cursor = "> "
			style = p.theme.Active
		}

		// FPM status display
		status := "stopped"
		statusStyle := p.theme.ErrorText
		if v.Active {
			status = "running"
			statusStyle = p.theme.OkText
		}

		// Boot enabled display
		boot := "disabled"
		bootStyle := p.theme.ErrorText
		if v.Enabled {
			boot = "enabled"
			bootStyle = p.theme.OkText
		}

		row := fmt.Sprintf("%s%-12s %s  %s",
			cursor,
			style.Render("PHP "+v.Version),
			statusStyle.Render(fmt.Sprintf("%-12s", status)),
			bootStyle.Render(boot),
		)
		rows += row + "\n"
	}

	help := p.theme.HelpDesc.Render("  i:install  r:remove  esc:back")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, "", headerStyle, rows, help)
}

func (p *PHPScreen) ScreenTitle() string { return "PHP" }

// Messages for PHP screen actions
type InstallPHPMsg struct{}

type RemovePHPMsg struct {
	Version string
}
