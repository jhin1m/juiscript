# Phase 3: Database Commands

## File: `cmd/juiscript/cmd-db.go`

### Command Tree

```
juiscript db list
juiscript db create --name mydb
juiscript db drop --name mydb
juiscript db user-create --username myuser --database mydb
juiscript db user-drop --username myuser
juiscript db reset-password --username myuser
juiscript db import --name mydb --file /path/to/dump.sql
juiscript db export --name mydb --output /path/to/dump.sql
```

### Implementation

```go
func dbCmd(mgrs *Managers) *cobra.Command {
    cmd := &cobra.Command{
        Use:     "db",
        Aliases: []string{"database"},
        Short:   "Manage databases and users",
    }
    cmd.AddCommand(
        dbListCmd(mgrs),
        dbCreateCmd(mgrs),
        dbDropCmd(mgrs),
        dbUserCreateCmd(mgrs),
        dbUserDropCmd(mgrs),
        dbResetPasswordCmd(mgrs),
        dbImportCmd(mgrs),
        dbExportCmd(mgrs),
    )
    return cmd
}
```

### Subcommand Details

**db list**
- Calls `mgrs.DB.ListDBs(ctx)`
- Output table: NAME | SIZE | TABLES

**db create --name X**
- Calls `mgrs.DB.CreateDB(ctx, name)`
- Print: "Database created: mydb"

**db drop --name X**
- Calls `mgrs.DB.DropDB(ctx, name)`
- Print: "Database dropped: mydb"

**db user-create --username X --database X**
- Calls `mgrs.DB.CreateUser(ctx, username, dbName)`
- Note: `CreateUser` returns (password, error). Print password to stdout.
- Print: "User created: myuser\nPassword: <generated>"
- Important: password output enables scripting (`juiscript db user-create ... | grep Password`)

**db user-drop --username X**
- Calls `mgrs.DB.DropUser(ctx, username)`
- Print: "User dropped: myuser"

**db reset-password --username X**
- Calls `mgrs.DB.ResetPassword(ctx, username)`
- Returns new password, print it.
- Print: "Password reset for: myuser\nPassword: <generated>"

**db import --name X --file /path**
- Calls `mgrs.DB.Import(ctx, name, filePath)`
- Supports .sql and .sql.gz (handled by manager)
- Print: "Import complete: mydb <- /path/to/dump.sql"

**db export --name X --output /path**
- Calls `mgrs.DB.Export(ctx, name, outputPath)`
- Supports .sql and .sql.gz (handled by manager)
- Print: "Export complete: mydb -> /path/to/dump.sql"

### Manager API Notes

- `CreateUser(ctx, username, dbName)` returns `(string, error)` -- second arg is dbName, not password
- `ResetPassword(ctx, username)` returns `(string, error)` -- auto-generates password
- `DropUser(ctx, username)` -- single username arg

## Acceptance Criteria

- [ ] All 8 database subcommands functional
- [ ] Password output for user-create and reset-password
- [ ] Import/export with .sql and .sql.gz support
- [ ] `go build` succeeds
