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

// handleCreateDB is a placeholder -- needs a form screen for name input.
func (a *App) handleCreateDB() tea.Cmd {
	if a.dbMgr == nil {
		return nil
	}
	return func() tea.Msg {
		return DBOpErrMsg{Err: fmt.Errorf("database creation requires a name input form (not yet implemented)")}
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

// handleImportDB is a placeholder -- needs a form screen for file path input.
func (a *App) handleImportDB(name string) tea.Cmd {
	if a.dbMgr == nil {
		return nil
	}
	return func() tea.Msg {
		return DBOpErrMsg{Err: fmt.Errorf("import requires a file path input form (not yet implemented)")}
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
