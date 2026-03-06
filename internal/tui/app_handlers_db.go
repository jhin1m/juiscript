package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// fetchDatabases loads the database list asynchronously.
func (a *App) fetchDatabases() tea.Cmd {
	if a.dbMgr == nil {
		return nil
	}
	return func() tea.Msg {
		dbs, err := a.dbMgr.ListDBs(context.Background())
		if err != nil {
			return DBListErrMsg{Err: err}
		}
		return DBListMsg{Databases: dbs}
	}
}

// handleCreateDB creates a database with the given name.
func (a *App) handleCreateDB(name string) tea.Cmd {
	if a.dbMgr == nil {
		return nil
	}
	return func() tea.Msg {
		if err := a.dbMgr.CreateDB(context.Background(), name); err != nil {
			return DBOpErrMsg{Err: err}
		}
		return DBOpDoneMsg{}
	}
}

// handleDropDB drops a database by name.
func (a *App) handleDropDB(name string) tea.Cmd {
	if a.dbMgr == nil {
		return nil
	}
	return func() tea.Msg {
		if err := a.dbMgr.DropDB(context.Background(), name); err != nil {
			return DBOpErrMsg{Err: err}
		}
		return DBOpDoneMsg{}
	}
}

// handleImportDB imports a SQL file into a database.
func (a *App) handleImportDB(name, path string) tea.Cmd {
	if a.dbMgr == nil {
		return nil
	}
	return func() tea.Msg {
		if err := a.dbMgr.Import(context.Background(), name, path); err != nil {
			return DBOpErrMsg{Err: err}
		}
		return DBOpDoneMsg{}
	}
}

// handleExportDB exports a database to a gzipped SQL file.
func (a *App) handleExportDB(name string) tea.Cmd {
	if a.dbMgr == nil {
		return nil
	}
	return func() tea.Msg {
		path := fmt.Sprintf("/var/backups/juiscript/%s.sql.gz", name)
		if err := a.dbMgr.Export(context.Background(), name, path); err != nil {
			return DBOpErrMsg{Err: err}
		}
		return DBOpDoneMsg{}
	}
}
