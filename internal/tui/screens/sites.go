package screens

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/site"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// SiteList displays all managed sites in a navigable table.
type SiteList struct {
	theme  *theme.Theme
	sites  []*site.Site
	cursor int
	width  int
	height int
	err    error
}

// NewSiteList creates the site listing screen.
func NewSiteList(t *theme.Theme) *SiteList {
	return &SiteList{
		theme: t,
	}
}

// SetSites updates the site data.
func (s *SiteList) SetSites(sites []*site.Site) {
	s.sites = sites
	s.err = nil
}

// SetError sets an error to display.
func (s *SiteList) SetError(err error) {
	s.err = err
}

func (s *SiteList) Init() tea.Cmd {
	return nil
}

func (s *SiteList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.cursor > 0 {
				s.cursor--
			}
		case "down", "j":
			if s.cursor < len(s.sites)-1 {
				s.cursor++
			}
		case "c":
			// Navigate to create form
			return s, func() tea.Msg {
				return ShowCreateFormMsg{}
			}
		case "enter":
			if len(s.sites) > 0 {
				return s, func() tea.Msg {
					return ShowSiteDetailMsg{Domain: s.sites[s.cursor].Domain}
				}
			}
		case "e":
			if len(s.sites) > 0 {
				return s, func() tea.Msg {
					return ToggleSiteMsg{Domain: s.sites[s.cursor].Domain}
				}
			}
		case "esc", "q":
			return s, func() tea.Msg { return GoBackMsg{} }
		}
	}

	return s, nil
}

func (s *SiteList) View() string {
	title := s.theme.Title.Render("Sites")

	if s.err != nil {
		errMsg := s.theme.ErrorText.Render(fmt.Sprintf("Error: %v", s.err))
		return lipgloss.JoinVertical(lipgloss.Left, title, "", errMsg)
	}

	if len(s.sites) == 0 {
		empty := s.theme.Subtitle.Render("  No sites configured. Press 'c' to create one.")
		return lipgloss.JoinVertical(lipgloss.Left, title, "", empty)
	}

	// Table header
	header := fmt.Sprintf("  %-30s %-12s %-6s %-5s %-8s",
		"DOMAIN", "TYPE", "PHP", "SSL", "STATUS")
	headerStyle := s.theme.HelpKey.Render(header)

	// Table rows
	var rows string
	for i, st := range s.sites {
		cursor := "  "
		style := s.theme.Inactive
		if i == s.cursor {
			cursor = "> "
			style = s.theme.Active
		}

		status := "active"
		statusStyle := s.theme.OkText
		if !st.Enabled {
			status = "disabled"
			statusStyle = s.theme.ErrorText
		}

		ssl := "no"
		if st.SSLEnabled {
			ssl = "yes"
		}

		row := fmt.Sprintf("%s%-30s %-12s %-6s %-5s %s",
			cursor,
			style.Render(st.Domain),
			string(st.ProjectType),
			st.PHPVersion,
			ssl,
			statusStyle.Render(status),
		)
		rows += row + "\n"
	}

	help := s.theme.HelpDesc.Render("  c:create  enter:detail  e:enable/disable  esc:back")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, "", headerStyle, rows, help)
}

func (s *SiteList) ScreenTitle() string { return "Sites" }

// Messages for site screen navigation
type ShowCreateFormMsg struct{}
type ShowSiteDetailMsg struct{ Domain string }
type ToggleSiteMsg struct{ Domain string }
