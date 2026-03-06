package screens

import (
	"fmt"
	"regexp"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/database"
	"github.com/jhin1m/juiscript/internal/tui/components"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// dbNameRegex validates database names: alphanumeric + underscores, 1-64 chars.
var dbNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1,64}$`)

func validateDBName(name string) error {
	if !dbNameRegex.MatchString(name) {
		return fmt.Errorf("alphanumeric and underscores only, 1-64 chars")
	}
	return nil
}

func validateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}

// DatabaseScreen displays databases and provides management actions.
type DatabaseScreen struct {
	theme  *theme.Theme
	dbs    []database.DBInfo
	cursor int
	width  int
	height int
	err    error
	// Form for create/import
	form       *components.FormModel
	formActive bool
	formAction string // "create" or "import"
	// Confirm for destructive drop action
	confirm       *components.ConfirmModel
	pendingTarget string // db name to drop
}

func NewDatabaseScreen(t *theme.Theme) *DatabaseScreen {
	return &DatabaseScreen{
		theme:   t,
		confirm: components.NewConfirm(t),
	}
}

func (d *DatabaseScreen) SetDatabases(dbs []database.DBInfo) {
	d.dbs = dbs
	d.err = nil
}

func (d *DatabaseScreen) SetError(err error) {
	d.err = err
}

func (d *DatabaseScreen) Init() tea.Cmd { return nil }

func (d *DatabaseScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Confirm dialog
	if d.confirm.Active() {
		_, cmd := d.confirm.Update(msg)
		if cmd != nil {
			name := d.pendingTarget
			d.pendingTarget = ""
			return d, func() tea.Msg {
				result := cmd()
				switch result.(type) {
				case components.ConfirmYesMsg:
					return DropDBMsg{Name: name}
				case components.ConfirmNoMsg:
					return nil
				default:
					return result
				}
			}
		}
		return d, nil
	}

	// Form
	if d.formActive {
		_, cmd := d.form.Update(msg)
		if cmd != nil {
			result := cmd()
			switch v := result.(type) {
			case components.FormSubmitMsg:
				d.formActive = false
				action := d.formAction
				switch action {
				case "create":
					return d, func() tea.Msg { return CreateDBMsg{Name: v.Values["name"]} }
				case "import":
					// Capture DB name synchronously to avoid cursor race
					dbName := d.dbs[d.cursor].Name
					path := v.Values["path"]
					return d, func() tea.Msg { return ImportDBMsg{Name: dbName, Path: path} }
				}
			case components.FormCancelMsg:
				d.formActive = false
				return d, nil
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
			if d.cursor < len(d.dbs)-1 {
				d.cursor++
			}
		case "c":
			// Create DB form
			fields := []components.FormField{
				{Key: "name", Label: "Database Name", Type: components.FieldText,
					Placeholder: "mydb", Validate: validateDBName},
			}
			d.form = components.NewForm(d.theme, "Create Database", fields)
			d.formActive = true
			d.formAction = "create"
		case "d":
			if len(d.dbs) > 0 {
				name := d.dbs[d.cursor].Name
				d.pendingTarget = name
				d.confirm.Show(fmt.Sprintf("Drop database '%s'? This cannot be undone.", name))
			}
		case "i":
			if len(d.dbs) > 0 {
				// Import form
				fields := []components.FormField{
					{Key: "path", Label: "SQL File Path", Type: components.FieldText,
						Placeholder: "/path/to/dump.sql.gz", Validate: validateFilePath},
				}
				d.form = components.NewForm(d.theme, fmt.Sprintf("Import into '%s'", d.dbs[d.cursor].Name), fields)
				d.formActive = true
				d.formAction = "import"
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

	// Confirm dialog
	if d.confirm.Active() {
		return lipgloss.JoinVertical(lipgloss.Left, title, "", d.confirm.View())
	}

	// Form
	if d.formActive && d.form != nil {
		return d.form.View()
	}

	if d.err != nil {
		errMsg := d.theme.ErrorText.Render(fmt.Sprintf("Error: %v", d.err))
		return lipgloss.JoinVertical(lipgloss.Left, title, "", errMsg)
	}

	if len(d.dbs) == 0 {
		empty := d.theme.Subtitle.Render("  No databases found.")
		help := d.theme.HelpDesc.Render("  c:create  esc:back")
		return lipgloss.JoinVertical(lipgloss.Left, title, "", empty, "", help)
	}

	header := fmt.Sprintf("  %-30s %-12s %s", "DATABASE", "SIZE (MB)", "TABLES")
	headerStyle := d.theme.HelpKey.Render(header)

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

func (d *DatabaseScreen) ScreenTitle() string { return "Database" }

// Messages for database screen actions.
type CreateDBMsg struct {
	Name string
}

type DropDBMsg struct {
	Name string
}

type ImportDBMsg struct {
	Name string
	Path string
}

type ExportDBMsg struct {
	Name string
}
