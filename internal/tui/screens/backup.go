package screens

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/backup"
	"github.com/jhin1m/juiscript/internal/site"
	"github.com/jhin1m/juiscript/internal/tui/components"
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
	// Form for create backup
	form       *components.FormModel
	formActive bool
	// Confirm for delete/restore
	confirm       *components.ConfirmModel
	pendingAction string // "delete" or "restore"
	pendingTarget string // path for delete/restore
	pendingDomain string // domain for restore
}

func NewBackupScreen(t *theme.Theme) *BackupScreen {
	return &BackupScreen{
		theme:   t,
		confirm: components.NewConfirm(t),
	}
}

func (b *BackupScreen) SetBackups(backups []backup.BackupInfo) {
	b.backups = backups
	b.err = nil
}

func (b *BackupScreen) SetError(err error) {
	b.err = err
}

// StopSpinner is a no-op kept for App compatibility.
func (b *BackupScreen) StopSpinner() {}

func (b *BackupScreen) Init() tea.Cmd { return nil }

func (b *BackupScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Confirm dialog
	if b.confirm.Active() {
		_, cmd := b.confirm.Update(msg)
		if cmd != nil {
			action := b.pendingAction
			target := b.pendingTarget
			domain := b.pendingDomain
			b.pendingAction = ""
			b.pendingTarget = ""
			b.pendingDomain = ""
			return b, func() tea.Msg {
				result := cmd()
				switch result.(type) {
				case components.ConfirmYesMsg:
					switch action {
					case "delete":
						return DeleteBackupMsg{Path: target}
					case "restore":
						return RestoreBackupMsg{Path: target, Domain: domain}
					}
				case components.ConfirmNoMsg:
					// cancelled
				}
				return nil
			}
		}
		return b, nil
	}

	// Form
	if b.formActive {
		_, cmd := b.form.Update(msg)
		if cmd != nil {
			result := cmd()
			switch v := result.(type) {
			case components.FormSubmitMsg:
				b.formActive = false
				domain := v.Values["domain"]
				backupType := v.Values["type"]
				// Don't block UI — backup runs in background
				return b, func() tea.Msg {
					return CreateBackupMsg{Domain: domain, Type: backupType}
				}
			case components.FormCancelMsg:
				b.formActive = false
				return b, nil
			}
		}
		return b, nil
	}

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
			// Create backup form
			fields := []components.FormField{
				{Key: "domain", Label: "Domain", Type: components.FieldText,
					Placeholder: "example.com", Validate: site.ValidateDomain},
				{Key: "type", Label: "Backup Type", Type: components.FieldSelect,
					Options: []string{"full", "files", "database"}, Default: "full"},
			}
			b.form = components.NewForm(b.theme, "Create Backup", fields)
			b.formActive = true
		case "r":
			// Restore - confirm first
			if len(b.backups) > 0 {
				bk := b.backups[b.cursor]
				b.pendingAction = "restore"
				b.pendingTarget = bk.Path
				b.pendingDomain = bk.Domain
				b.confirm.Show(fmt.Sprintf("Restore backup for '%s'? Current data will be overwritten.", bk.Domain))
			}
		case "d":
			// Delete - confirm first
			if len(b.backups) > 0 {
				bk := b.backups[b.cursor]
				b.pendingAction = "delete"
				b.pendingTarget = bk.Path
				b.confirm.Show(fmt.Sprintf("Delete backup '%s'? This cannot be undone.", bk.Domain))
			}
		case "esc", "q":
			return b, func() tea.Msg { return GoBackMsg{} }
		}
	}

	return b, nil
}

func (b *BackupScreen) View() string {
	title := b.theme.Title.Render("Backups")

	if b.confirm.Active() {
		return lipgloss.JoinVertical(lipgloss.Left, title, "", b.confirm.View())
	}

	if b.formActive && b.form != nil {
		return b.form.View()
	}

	if b.err != nil {
		errMsg := b.theme.ErrorText.Render(fmt.Sprintf("Error: %v", b.err))
		return lipgloss.JoinVertical(lipgloss.Left, title, "", errMsg)
	}

	if len(b.backups) == 0 {
		empty := b.theme.Subtitle.Render("  No backups found. Press 'c' to create one.")
		help := b.theme.HelpDesc.Render("  c:create  esc:back")
		return lipgloss.JoinVertical(lipgloss.Left, title, "", empty, "", help)
	}

	header := fmt.Sprintf("  %-30s %-12s %-20s", "DOMAIN", "SIZE", "CREATED")
	headerStyle := b.theme.HelpKey.Render(header)

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
type CreateBackupMsg struct {
	Domain string
	Type   string
}

type RestoreBackupMsg struct {
	Path   string
	Domain string
}

type DeleteBackupMsg struct {
	Path string
}
