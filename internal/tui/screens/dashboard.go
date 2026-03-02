package screens

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// MenuItem represents a dashboard menu option.
type MenuItem struct {
	Title string
	Desc  string
	Key   string
}

// Dashboard is the main screen showing system overview and navigation.
type Dashboard struct {
	theme        *theme.Theme
	items        []MenuItem
	cursor       int
	missingCount int // number of missing LEMP packages
	width        int
	height       int
}

func NewDashboard(t *theme.Theme) *Dashboard {
	return &Dashboard{
		theme: t,
		items: []MenuItem{
			{Title: "Sites", Desc: "Manage websites and domains", Key: "1"},
			{Title: "Nginx", Desc: "Virtual host configuration", Key: "2"},
			{Title: "PHP", Desc: "PHP versions and FPM pools", Key: "3"},
			{Title: "Database", Desc: "MariaDB databases and users", Key: "4"},
			{Title: "SSL", Desc: "Let's Encrypt certificates", Key: "5"},
			{Title: "Services", Desc: "Start/stop/restart services", Key: "6"},
			{Title: "Queues", Desc: "Supervisor queue workers", Key: "7"},
			{Title: "Backup", Desc: "Backup and restore sites", Key: "8"},
			{Title: "Setup", Desc: "Install missing packages", Key: "9"},
		},
	}
}

func (d *Dashboard) Init() tea.Cmd {
	return nil
}

func (d *Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if d.cursor < len(d.items)-1 {
				d.cursor++
			}
		case "enter":
			return d, func() tea.Msg {
				return NavigateMsg{Screen: d.items[d.cursor].Title}
			}
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(msg.String()[0] - '1')
			if idx < len(d.items) {
				return d, func() tea.Msg {
					return NavigateMsg{Screen: d.items[idx].Title}
				}
			}
		}
	}

	return d, nil
}

func (d *Dashboard) View() string {
	// Title section
	title := d.theme.Title.Render("Dashboard")
	subtitle := d.theme.Subtitle.Render("LEMP Server Management")
	header := lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "")

	// Warning banner for missing packages
	var banner string
	if d.missingCount > 0 {
		banner = d.theme.WarnText.Render(
			fmt.Sprintf("  ⚠ %d package(s) not installed — press '9' for Setup", d.missingCount)) + "\n"
	}

	// Menu items
	var menuItems string
	for i, item := range d.items {
		cursor := "  "
		style := d.theme.Inactive
		if i == d.cursor {
			cursor = "> "
			style = d.theme.Active
		}

		key := d.theme.HelpKey.Render(fmt.Sprintf("[%s]", item.Key))
		title := style.Render(item.Title)
		desc := d.theme.Subtitle.Render(item.Desc)

		line := fmt.Sprintf("%s%s %s  %s", cursor, key, title, desc)
		menuItems += line + "\n"
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, banner, menuItems)
}

// ScreenTitle returns the title for the header component.
func (d *Dashboard) ScreenTitle() string {
	return "Dashboard"
}

// SetMissingCount updates the count of missing packages for the warning banner.
func (d *Dashboard) SetMissingCount(n int) {
	d.missingCount = n
}

// NavigateMsg signals the root model to switch screens.
type NavigateMsg struct {
	Screen string
}

// GoBackMsg signals the root model to go back to dashboard.
type GoBackMsg struct{}
