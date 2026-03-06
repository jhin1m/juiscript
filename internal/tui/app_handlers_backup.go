package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jhin1m/juiscript/internal/backup"
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

// handleCreateBackup creates a backup for a domain with the given type.
func (a *App) handleCreateBackup(domain, backupType string) tea.Cmd {
	if a.backupMgr == nil {
		return nil
	}
	return func() tea.Msg {
		opts := backup.Options{
			Domain: domain,
			Type:   backup.BackupType(backupType),
		}
		if _, err := a.backupMgr.Create(context.Background(), opts); err != nil {
			return BackupOpErrMsg{Err: err}
		}
		return BackupOpDoneMsg{}
	}
}

// handleRestoreBackup restores a backup to a domain.
func (a *App) handleRestoreBackup(path, domain string) tea.Cmd {
	if a.backupMgr == nil {
		return nil
	}
	return func() tea.Msg {
		if err := a.backupMgr.Restore(context.Background(), path, domain); err != nil {
			return BackupOpErrMsg{Err: err}
		}
		return BackupOpDoneMsg{}
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
