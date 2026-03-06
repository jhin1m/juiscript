package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

// handleInstallPHP installs a PHP version and re-fetches the version list on success.
func (a *App) handleInstallPHP(version string) tea.Cmd {
	if a.phpMgr == nil {
		return nil
	}
	return func() tea.Msg {
		if err := a.phpMgr.InstallVersion(context.Background(), version); err != nil {
			return PHPVersionsErrMsg{Err: err}
		}
		// Re-fetch versions after install completes
		versions, err := a.phpMgr.ListVersions(context.Background())
		if err != nil {
			return PHPVersionsErrMsg{Err: err}
		}
		return PHPVersionsMsg{Versions: versions}
	}
}

// handleRemovePHP removes a PHP version and re-fetches the version list on success.
// Passes nil for sites list -- TUI doesn't track site-version mapping yet.
func (a *App) handleRemovePHP(version string) tea.Cmd {
	if a.phpMgr == nil {
		return nil
	}
	return func() tea.Msg {
		if err := a.phpMgr.RemoveVersion(context.Background(), version, nil); err != nil {
			return PHPVersionsErrMsg{Err: err}
		}
		versions, err := a.phpMgr.ListVersions(context.Background())
		if err != nil {
			return PHPVersionsErrMsg{Err: err}
		}
		return PHPVersionsMsg{Versions: versions}
	}
}
