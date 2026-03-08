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
		return PHPVersionsMsg{Versions: versions, Action: "install"}
	}
}

// handleRemovePHP removes a PHP version and re-fetches the version list on success.
// Fetches active sites first so RemoveVersion can warn if the version is in use.
func (a *App) handleRemovePHP(version string) tea.Cmd {
	if a.phpMgr == nil {
		return nil
	}
	return func() tea.Msg {
		// Collect domains of sites using this PHP version
		var activeSites []string
		if a.siteMgr != nil {
			if sites, err := a.siteMgr.List(); err == nil {
				for _, s := range sites {
					if s.PHPVersion == version {
						activeSites = append(activeSites, s.Domain)
					}
				}
			}
		}
		if err := a.phpMgr.RemoveVersion(context.Background(), version, activeSites); err != nil {
			return PHPVersionsErrMsg{Err: err}
		}
		versions, err := a.phpMgr.ListVersions(context.Background())
		if err != nil {
			return PHPVersionsErrMsg{Err: err}
		}
		return PHPVersionsMsg{Versions: versions, Action: "remove"}
	}
}
