package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

func (a *App) handleOpenPort(port int, proto string) tea.Cmd {
	if a.firewallMgr == nil {
		return nil
	}
	return func() tea.Msg {
		err := a.firewallMgr.AllowPort(context.Background(), port, proto)
		if err != nil {
			return FirewallOpErrMsg{Err: err}
		}
		return FirewallOpDoneMsg{}
	}
}

func (a *App) handleClosePort(port int, proto string) tea.Cmd {
	if a.firewallMgr == nil {
		return nil
	}
	return func() tea.Msg {
		err := a.firewallMgr.DenyPort(context.Background(), port, proto)
		if err != nil {
			return FirewallOpErrMsg{Err: err}
		}
		return FirewallOpDoneMsg{}
	}
}

func (a *App) handleDeleteUFWRule(ruleNum int) tea.Cmd {
	if a.firewallMgr == nil {
		return nil
	}
	return func() tea.Msg {
		err := a.firewallMgr.DeleteRule(context.Background(), ruleNum)
		if err != nil {
			return FirewallOpErrMsg{Err: err}
		}
		return FirewallOpDoneMsg{}
	}
}

func (a *App) handleBanIP(ip, jail string) tea.Cmd {
	if a.firewallMgr == nil {
		return nil
	}
	return func() tea.Msg {
		err := a.firewallMgr.BanIP(context.Background(), ip, jail)
		if err != nil {
			return FirewallOpErrMsg{Err: err}
		}
		return FirewallOpDoneMsg{}
	}
}

func (a *App) handleUnbanIP(ip, jail string) tea.Cmd {
	if a.firewallMgr == nil {
		return nil
	}
	return func() tea.Msg {
		err := a.firewallMgr.UnbanIP(context.Background(), ip, jail)
		if err != nil {
			return FirewallOpErrMsg{Err: err}
		}
		return FirewallOpDoneMsg{}
	}
}
