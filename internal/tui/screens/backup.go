package screens

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/backup"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// BackupScreen displays backup management for sites.
type BackupScreen struct {
	theme   *theme.Theme
	backups []backup.BackupInfo
	cursor  int
	width   int
	height  int
	err     error
}

// NewBackupScreen creates the backup management screen.
func NewBackupScreen(t *theme.Theme) *BackupScreen {
	return &BackupScreen{theme: t}
}

// SetBackups updates the backup list.
func (b *BackupScreen) SetBackups(backups []backup.BackupInfo) {
	b.backups = backups
	b.err = nil
}

// SetError sets an error to display.
func (b *BackupScreen) SetError(err error) {
	b.err = err
}

func (b *BackupScreen) Init() tea.Cmd { return nil }

func (b *BackupScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if b.cursor > 0 {
				b.cursor--
			}
		case "down", "j":
			if b.cursor < len(b.backups)-1 {
				b.cursor++
			}
		case "c":
			// Create new backup
			return b, func() tea.Msg { return CreateBackupMsg{} }
		case "r":
			// Restore selected backup
			if len(b.backups) > 0 {
				return b, func() tea.Msg {
					return RestoreBackupMsg{Path: b.backups[b.cursor].Path}
				}
			}
		case "d":
			// Delete selected backup
			if len(b.backups) > 0 {
				return b, func() tea.Msg {
					return DeleteBackupMsg{Path: b.backups[b.cursor].Path}
				}
			}
		case "esc", "q":
			return b, func() tea.Msg { return GoBackMsg{} }
		}
	}

	return b, nil
}

func (b *BackupScreen) View() string {
	title := b.theme.Title.Render("Backups")

	if b.err != nil {
		errMsg := b.theme.ErrorText.Render(fmt.Sprintf("Error: %v", b.err))
		return lipgloss.JoinVertical(lipgloss.Left, title, "", errMsg)
	}

	if len(b.backups) == 0 {
		empty := b.theme.Subtitle.Render("  No backups found. Press 'c' to create one.")
		help := b.theme.HelpDesc.Render("  c:create  esc:back")
		return lipgloss.JoinVertical(lipgloss.Left, title, "", empty, "", help)
	}

	// Table header
	header := fmt.Sprintf("  %-30s %-12s %-20s", "DOMAIN", "SIZE", "CREATED")
	headerStyle := b.theme.HelpKey.Render(header)

	// Table rows
	var rows string
	for i, bk := range b.backups {
		cursor := "  "
		style := b.theme.Inactive
		if i == b.cursor {
			cursor = "> "
			style = b.theme.Active
		}

		row := fmt.Sprintf("%s%-30s %-12s %-20s",
			cursor,
			style.Render(bk.Domain),
			b.theme.Subtitle.Render(backup.FormatSize(bk.Size)),
			b.theme.Subtitle.Render(bk.CreatedAt.Format("2006-01-02 15:04:05")),
		)
		rows += row + "\n"
	}

	help := b.theme.HelpDesc.Render("  c:create  r:restore  d:delete  esc:back")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, "", headerStyle, rows, help)
}

func (b *BackupScreen) ScreenTitle() string { return "Backup" }

// Messages for backup screen actions
type CreateBackupMsg struct{}

type RestoreBackupMsg struct {
	Path string
}

type DeleteBackupMsg struct {
	Path string
}
