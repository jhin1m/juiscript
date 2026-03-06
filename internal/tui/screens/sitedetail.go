package screens

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/site"
	"github.com/jhin1m/juiscript/internal/tui/components"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// SiteDetail shows detailed info about a single site with action menu.
type SiteDetail struct {
	theme   *theme.Theme
	site    *site.Site
	cursor  int
	width   int
	height  int
	confirm *components.ConfirmModel
}

type detailAction struct {
	Key  string
	Name string
}

var detailActions = []detailAction{
	{Key: "e", Name: "Enable/Disable"},
	{Key: "d", Name: "Delete Site"},
}

func NewSiteDetail(t *theme.Theme) *SiteDetail {
	return &SiteDetail{
		theme:   t,
		confirm: components.NewConfirm(t),
	}
}

func (d *SiteDetail) SetSite(s *site.Site) {
	d.site = s
	d.cursor = 0
}

func (d *SiteDetail) Init() tea.Cmd { return nil }

func (d *SiteDetail) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Confirm dialog
	if d.confirm.Active() {
		_, cmd := d.confirm.Update(msg)
		if cmd != nil {
			return d, func() tea.Msg {
				result := cmd()
				switch result.(type) {
				case components.ConfirmYesMsg:
					if d.site != nil {
						return DeleteSiteMsg{Domain: d.site.Domain}
					}
				case components.ConfirmNoMsg:
					// cancelled
				}
				return nil
			}
		}
		return d, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if d.cursor > 0 {
				d.cursor--
			}
		case "down", "j":
			if d.cursor < len(detailActions)-1 {
				d.cursor++
			}
		case "e":
			if d.site != nil {
				return d, func() tea.Msg {
					return ToggleSiteMsg{Domain: d.site.Domain}
				}
			}
		case "d":
			if d.site != nil {
				d.confirm.Show(fmt.Sprintf("Delete site '%s'? This will remove all files and configs.", d.site.Domain))
			}
		case "esc", "q":
			return d, func() tea.Msg { return GoBackMsg{} }
		}
	}

	return d, nil
}

func (d *SiteDetail) View() string {
	if d.site == nil {
		return d.theme.Subtitle.Render("  No site selected")
	}

	s := d.site
	title := d.theme.Title.Render(s.Domain)

	// Confirm dialog replaces content
	if d.confirm.Active() {
		return lipgloss.JoinVertical(lipgloss.Left, title, "", d.confirm.View())
	}

	status := d.theme.OkText.Render("ACTIVE")
	if !s.Enabled {
		status = d.theme.ErrorText.Render("DISABLED")
	}

	info := fmt.Sprintf(
		"  Status:       %s\n"+
			"  Type:         %s\n"+
			"  PHP Version:  %s\n"+
			"  User:         %s\n"+
			"  Web Root:     %s\n"+
			"  SSL:          %s\n"+
			"  Database:     %s\n"+
			"  Created:      %s",
		status,
		s.ProjectType,
		s.PHPVersion,
		s.User,
		s.WebRoot,
		boolToYesNo(s.SSLEnabled),
		nonEmpty(s.DBName, "none"),
		s.CreatedAt.Format("2006-01-02 15:04"),
	)

	actions := "\n  Actions:\n"
	for i, a := range detailActions {
		cursor := "  "
		if i == d.cursor {
			cursor = "> "
		}
		key := d.theme.HelpKey.Render(fmt.Sprintf("[%s]", a.Key))
		actions += fmt.Sprintf("  %s%s %s\n", cursor, key, a.Name)
	}

	help := d.theme.HelpDesc.Render("  esc:back")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, "", info, actions, help)
}

func (d *SiteDetail) ScreenTitle() string {
	if d.site != nil {
		return d.site.Domain
	}
	return "Site Detail"
}

// DeleteSiteMsg signals the app to delete a site.
type DeleteSiteMsg struct{ Domain string }

func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func nonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
