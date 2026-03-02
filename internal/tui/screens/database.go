package screens

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/database"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// DatabaseScreen displays databases and provides management actions.
type DatabaseScreen struct {
	theme  *theme.Theme
	dbs    []database.DBInfo
	cursor int
	width  int
	height int
	err    error
}

// NewDatabaseScreen creates the database management screen.
func NewDatabaseScreen(t *theme.Theme) *DatabaseScreen {
	return &DatabaseScreen{theme: t}
}

// SetDatabases updates the database list.
func (d *DatabaseScreen) SetDatabases(dbs []database.DBInfo) {
	d.dbs = dbs
	d.err = nil
}

// SetError sets an error to display.
func (d *DatabaseScreen) SetError(err error) {
	d.err = err
}

func (d *DatabaseScreen) Init() tea.Cmd { return nil }

func (d *DatabaseScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if d.cursor < len(d.dbs)-1 {
				d.cursor++
			}
		case "c":
			return d, func() tea.Msg { return CreateDBMsg{} }
		case "d":
			if len(d.dbs) > 0 {
				return d, func() tea.Msg {
					return DropDBMsg{Name: d.dbs[d.cursor].Name}
				}
			}
		case "i":
			if len(d.dbs) > 0 {
				return d, func() tea.Msg {
					return ImportDBMsg{Name: d.dbs[d.cursor].Name}
				}
			}
		case "e":
			if len(d.dbs) > 0 {
				return d, func() tea.Msg {
					return ExportDBMsg{Name: d.dbs[d.cursor].Name}
				}
			}
		case "esc", "q":
			return d, func() tea.Msg { return GoBackMsg{} }
		}
	}

	return d, nil
}

func (d *DatabaseScreen) View() string {
	title := d.theme.Title.Render("Databases")

	if d.err != nil {
		errMsg := d.theme.ErrorText.Render(fmt.Sprintf("Error: %v", d.err))
		return lipgloss.JoinVertical(lipgloss.Left, title, "", errMsg)
	}

	if len(d.dbs) == 0 {
		empty := d.theme.Subtitle.Render("  No databases found.")
		help := d.theme.HelpDesc.Render("  c:create  esc:back")
		return lipgloss.JoinVertical(lipgloss.Left, title, "", empty, "", help)
	}

	// Table header
	header := fmt.Sprintf("  %-30s %-12s %s", "DATABASE", "SIZE (MB)", "TABLES")
	headerStyle := d.theme.HelpKey.Render(header)

	// Table rows
	var rows string
	for idx, db := range d.dbs {
		cursor := "  "
		style := d.theme.Inactive
		if idx == d.cursor {
			cursor = "> "
			style = d.theme.Active
		}

		row := fmt.Sprintf("%s%-30s %-12.2f %d",
			cursor,
			style.Render(db.Name),
			db.SizeMB,
			db.Tables,
		)
		rows += row + "\n"
	}

	help := d.theme.HelpDesc.Render("  c:create  d:drop  i:import  e:export  esc:back")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, "", headerStyle, rows, help)
}

// ScreenTitle returns the title for the header component.
func (d *DatabaseScreen) ScreenTitle() string { return "Database" }

// Messages for database screen actions.
type CreateDBMsg struct{}

type DropDBMsg struct {
	Name string
}

type ImportDBMsg struct {
	Name string
}

type ExportDBMsg struct {
	Name string
}
