package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jhin1m/juiscript/internal/site"
)

// fetchSites loads the site list asynchronously for the SiteList screen.
func (a *App) fetchSites() tea.Cmd {
	if a.siteMgr == nil {
		return nil
	}
	return func() tea.Msg {
		sites, err := a.siteMgr.List()
		if err != nil {
			return SiteListErrMsg{Err: err}
		}
		return SiteListMsg{Sites: sites}
	}
}

// fetchSiteDetail loads a single site's data for the SiteDetail screen.
func (a *App) fetchSiteDetail(domain string) tea.Cmd {
	if a.siteMgr == nil {
		return nil
	}
	return func() tea.Msg {
		s, err := a.siteMgr.Get(domain)
		if err != nil {
			return SiteOpErrMsg{Err: err}
		}
		return SiteDetailMsg{Site: s}
	}
}

// handleCreateSite creates a new site asynchronously.
func (a *App) handleCreateSite(opts site.CreateOptions) tea.Cmd {
	if a.siteMgr == nil {
		return nil
	}
	return func() tea.Msg {
		s, err := a.siteMgr.Create(opts)
		if err != nil {
			return SiteOpErrMsg{Err: err}
		}
		return SiteCreatedMsg{Site: s}
	}
}

// handleToggleSite enables or disables a site based on its current state.
func (a *App) handleToggleSite(domain string) tea.Cmd {
	if a.siteMgr == nil {
		return nil
	}
	return func() tea.Msg {
		s, err := a.siteMgr.Get(domain)
		if err != nil {
			return SiteOpErrMsg{Err: err}
		}
		if s.Enabled {
			err = a.siteMgr.Disable(domain)
		} else {
			err = a.siteMgr.Enable(domain)
		}
		if err != nil {
			return SiteOpErrMsg{Err: err}
		}
		return SiteOpDoneMsg{}
	}
}

// handleDeleteSite removes a site. removeDB=false for now (no confirmation dialog).
func (a *App) handleDeleteSite(domain string) tea.Cmd {
	if a.siteMgr == nil {
		return nil
	}
	return func() tea.Msg {
		if err := a.siteMgr.Delete(domain, false); err != nil {
			return SiteOpErrMsg{Err: err}
		}
		return SiteOpDoneMsg{}
	}
}
