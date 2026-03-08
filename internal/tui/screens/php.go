package screens

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/php"
	"github.com/jhin1m/juiscript/internal/tui/components"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// Available PHP versions for installation.
var availablePHPVersions = []string{"8.4", "8.3", "8.2", "8.1", "8.0", "7.4"}

// PHPScreen displays installed PHP versions and their FPM status.
type PHPScreen struct {
	theme          *theme.Theme
	versions       []php.VersionInfo
	defaultVersion string
	cursor         int
	width          int
	height         int
	err            error
	// Form for version picker
	form       *components.FormModel
	formActive bool
	// Confirm for destructive remove action
	confirm       *components.ConfirmModel
	pendingTarget string // version to remove
}

func NewPHPScreen(t *theme.Theme) *PHPScreen {
	return &PHPScreen{
		theme:   t,
		confirm: components.NewConfirm(t),
	}
}

func (p *PHPScreen) SetVersions(versions []php.VersionInfo) {
	p.versions = versions
	p.err = nil
}

func (p *PHPScreen) SetDefaultVersion(ver string) {
	p.defaultVersion = ver
}

func (p *PHPScreen) SetError(err error) {
	p.err = err
}

// StopSpinner is a no-op kept for App compatibility.
// Install/remove now run in background without blocking the UI.
func (p *PHPScreen) StopSpinner() {}

func (p *PHPScreen) Init() tea.Cmd { return nil }

func (p *PHPScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Confirm dialog takes priority
	if p.confirm.Active() {
		_, cmd := p.confirm.Update(msg)
		if cmd != nil {
			ver := p.pendingTarget
			p.pendingTarget = ""
			return p, func() tea.Msg {
				result := cmd()
				switch result.(type) {
				case components.ConfirmYesMsg:
					return RemovePHPMsg{Version: ver}
				case components.ConfirmNoMsg:
					return nil
				default:
					return result
				}
			}
		}
		return p, nil
	}

	// Form takes priority when active
	if p.formActive {
		_, cmd := p.form.Update(msg)
		if cmd != nil {
			// Eagerly evaluate to handle form results synchronously
			result := cmd()
			switch v := result.(type) {
			case components.FormSubmitMsg:
				p.formActive = false
				version := v.Values["version"]
				// Don't block UI with spinner — install runs in background
				return p, func() tea.Msg {
					return InstallPHPMsg{Version: version}
				}
			case components.FormCancelMsg:
				p.formActive = false
				return p, nil
			}
		}
		return p, nil
	}

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
		case "d":
			if len(p.versions) > 0 {
				ver := p.versions[p.cursor].Version
				return p, func() tea.Msg {
					return SetDefaultPHPMsg{Version: ver}
				}
			}
		case "i":
			// Show version picker form
			fields := []components.FormField{
				{Key: "version", Label: "PHP Version", Type: components.FieldSelect,
					Options: availablePHPVersions, Default: availablePHPVersions[0]},
			}
			p.form = components.NewForm(p.theme, "Install PHP Version", fields)
			p.formActive = true
		case "r":
			if len(p.versions) > 0 {
				ver := p.versions[p.cursor].Version
				p.pendingTarget = ver
				p.confirm.Show(fmt.Sprintf("Remove PHP %s? Sites using it will break.", ver))
			}
		case "esc", "q":
			return p, func() tea.Msg { return GoBackMsg{} }
		}
	}

	return p, nil
}

func (p *PHPScreen) View() string {
	title := p.theme.Title.Render("PHP Versions")

	// Confirm dialog replaces content
	if p.confirm.Active() {
		return lipgloss.JoinVertical(lipgloss.Left, title, "", p.confirm.View())
	}

	// Form replaces content
	if p.formActive && p.form != nil {
		return p.form.View()
	}

	if p.err != nil {
		errMsg := p.theme.ErrorText.Render(fmt.Sprintf("Error: %v", p.err))
		return lipgloss.JoinVertical(lipgloss.Left, title, "", errMsg)
	}

	if len(p.versions) == 0 {
		empty := p.theme.Subtitle.Render("  No PHP versions installed.")
		help := p.theme.HelpDesc.Render("  i:install  esc:back")
		return lipgloss.JoinVertical(lipgloss.Left, title, "", empty, "", help)
	}

	header := fmt.Sprintf("  %-12s %-12s %-10s %s", "VERSION", "FPM STATUS", "BOOT", "")
	headerStyle := p.theme.HelpKey.Render(header)

	var rows string
	for i, v := range p.versions {
		cursor := "  "
		style := p.theme.Inactive
		if i == p.cursor {
			cursor = "> "
			style = p.theme.Active
		}

		status := "stopped"
		statusStyle := p.theme.ErrorText
		if v.Active {
			status = "running"
			statusStyle = p.theme.OkText
		}

		boot := "disabled"
		bootStyle := p.theme.ErrorText
		if v.Enabled {
			boot = "enabled"
			bootStyle = p.theme.OkText
		}

		defaultTag := ""
		if v.Version == p.defaultVersion {
			defaultTag = p.theme.OkText.Render(" ★ default")
		}

		row := fmt.Sprintf("%s%-12s %s  %s%s",
			cursor,
			style.Render("PHP "+v.Version),
			statusStyle.Render(fmt.Sprintf("%-12s", status)),
			bootStyle.Render(boot),
			defaultTag,
		)
		rows += row + "\n"
	}

	help := p.theme.HelpDesc.Render("  d:set default  i:install  r:remove  esc:back")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, "", headerStyle, rows, help)
}

func (p *PHPScreen) ScreenTitle() string { return "PHP" }

// Messages for PHP screen actions
type InstallPHPMsg struct {
	Version string
}

type RemovePHPMsg struct {
	Version string
}

type SetDefaultPHPMsg struct {
	Version string
}
