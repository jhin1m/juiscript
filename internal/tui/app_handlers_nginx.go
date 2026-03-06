package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// fetchVhosts loads the vhost list asynchronously for the Nginx screen.
func (a *App) fetchVhosts() tea.Cmd {
	if a.nginxMgr == nil {
		return nil
	}
	return func() tea.Msg {
		vhosts, err := a.nginxMgr.List()
		if err != nil {
			return VhostListErrMsg{Err: err}
		}
		return VhostListMsg{Vhosts: vhosts}
	}
}

// handleToggleVhost enables or disables a vhost based on its current state.
func (a *App) handleToggleVhost(domain string, currentlyEnabled bool) tea.Cmd {
	if a.nginxMgr == nil {
		return nil
	}
	return func() tea.Msg {
		var err error
		if currentlyEnabled {
			err = a.nginxMgr.Disable(domain)
		} else {
			err = a.nginxMgr.Enable(domain)
		}
		if err != nil {
			return NginxOpErrMsg{Err: err}
		}
		return NginxOpDoneMsg{}
	}
}

// handleDeleteVhost removes a vhost configuration.
func (a *App) handleDeleteVhost(domain string) tea.Cmd {
	if a.nginxMgr == nil {
		return nil
	}
	return func() tea.Msg {
		if err := a.nginxMgr.Delete(domain); err != nil {
			return NginxOpErrMsg{Err: err}
		}
		return NginxOpDoneMsg{}
	}
}

// handleTestNginx runs nginx configuration test.
func (a *App) handleTestNginx() tea.Cmd {
	if a.nginxMgr == nil {
		return nil
	}
	return func() tea.Msg {
		if err := a.nginxMgr.Test(); err != nil {
			return NginxOpErrMsg{Err: err}
		}
		return NginxTestOkMsg{Output: "nginx: configuration test passed"}
	}
}
