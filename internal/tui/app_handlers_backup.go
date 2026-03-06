package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// fetchBackups loads the backup list for a domain (empty string lists all).
func (a *App) fetchBackups(domain string) tea.Cmd {
	if a.backupMgr == nil {
		return nil
	}
	return func() tea.Msg {
		backups, err := a.backupMgr.List(domain)
		if err != nil {
			return BackupListErrMsg{Err: err}
		}
		return BackupListMsg{Backups: backups}
	}
}

// handleCreateBackup is a placeholder -- needs domain + type selection form.
func (a *App) handleCreateBackup() tea.Cmd {
	if a.backupMgr == nil {
		return nil
	}
	return func() tea.Msg {
		return BackupOpErrMsg{Err: fmt.Errorf("backup creation requires domain and type selection (not yet implemented)")}
	}
}

// handleRestoreBackup is a placeholder -- needs domain context for restore.
func (a *App) handleRestoreBackup(path string) tea.Cmd {
	if a.backupMgr == nil {
		return nil
	}
	return func() tea.Msg {
		return BackupOpErrMsg{Err: fmt.Errorf("restore requires domain context (not yet implemented)")}
	}
}

// handleDeleteBackup deletes a backup file by path.
func (a *App) handleDeleteBackup(path string) tea.Cmd {
	if a.backupMgr == nil {
		return nil
	}
	return func() tea.Msg {
		if err := a.backupMgr.Delete(path); err != nil {
			return BackupOpErrMsg{Err: err}
		}
		return BackupOpDoneMsg{}
	}
}
