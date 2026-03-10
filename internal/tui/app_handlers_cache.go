package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

func (a *App) handleFlushRedisDB(db int) tea.Cmd {
	if a.cacheMgr == nil {
		return nil
	}
	return func() tea.Msg {
		err := a.cacheMgr.FlushDB(context.Background(), db)
		if err != nil {
			return CacheOpErrMsg{Err: err}
		}
		return CacheOpDoneMsg{}
	}
}

func (a *App) handleFlushRedisAll() tea.Cmd {
	if a.cacheMgr == nil {
		return nil
	}
	return func() tea.Msg {
		err := a.cacheMgr.FlushAll(context.Background())
		if err != nil {
			return CacheOpErrMsg{Err: err}
		}
		return CacheOpDoneMsg{}
	}
}

func (a *App) handleResetOpcache(phpVersion string) tea.Cmd {
	if a.cacheMgr == nil {
		return nil
	}
	return func() tea.Msg {
		err := a.cacheMgr.ResetOpcache(context.Background(), phpVersion)
		if err != nil {
			return CacheOpErrMsg{Err: err}
		}
		return CacheOpDoneMsg{}
	}
}
