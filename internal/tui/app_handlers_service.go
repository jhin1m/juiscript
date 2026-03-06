package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jhin1m/juiscript/internal/service"
)

// handleServiceAction consolidates start/stop/restart/reload into one method (DRY).
// After the action, it re-fetches service statuses for immediate UI refresh.
func (a *App) handleServiceAction(name service.ServiceName, action string) tea.Cmd {
	if a.svcMgr == nil {
		return nil
	}
	return func() tea.Msg {
		ctx := context.Background()
		var err error
		switch action {
		case "start":
			err = a.svcMgr.Start(ctx, name)
		case "stop":
			err = a.svcMgr.Stop(ctx, name)
		case "restart":
			err = a.svcMgr.Restart(ctx, name)
		case "reload":
			err = a.svcMgr.Reload(ctx, name)
		}
		if err != nil {
			return ServiceOpErrMsg{Err: err}
		}
		// Re-fetch statuses after action for immediate UI update
		statuses, err := a.svcMgr.ListAll(ctx)
		if err != nil {
			return ServiceStatusErrMsg{Err: err}
		}
		return ServiceStatusMsg{Services: statuses}
	}
}
