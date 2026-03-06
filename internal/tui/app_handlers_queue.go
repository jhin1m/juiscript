package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

// fetchWorkers loads the supervisor worker list asynchronously.
func (a *App) fetchWorkers() tea.Cmd {
	if a.supervisorMgr == nil {
		return nil
	}
	return func() tea.Msg {
		workers, err := a.supervisorMgr.ListAll(context.Background())
		if err != nil {
			return WorkerListErrMsg{Err: err}
		}
		return WorkerListMsg{Workers: workers}
	}
}

// handleWorkerAction consolidates start/stop/restart into one method (DRY).
func (a *App) handleWorkerAction(name string, action string) tea.Cmd {
	if a.supervisorMgr == nil {
		return nil
	}
	return func() tea.Msg {
		ctx := context.Background()
		var err error
		switch action {
		case "start":
			err = a.supervisorMgr.Start(ctx, name)
		case "stop":
			err = a.supervisorMgr.Stop(ctx, name)
		case "restart":
			err = a.supervisorMgr.Restart(ctx, name)
		}
		if err != nil {
			return QueueOpErrMsg{Err: err}
		}
		return QueueOpDoneMsg{}
	}
}

// handleDeleteWorker removes a supervisor worker configuration.
func (a *App) handleDeleteWorker(name string) tea.Cmd {
	if a.supervisorMgr == nil {
		return nil
	}
	return func() tea.Msg {
		if err := a.supervisorMgr.Delete(context.Background(), name); err != nil {
			return QueueOpErrMsg{Err: err}
		}
		return QueueOpDoneMsg{}
	}
}
