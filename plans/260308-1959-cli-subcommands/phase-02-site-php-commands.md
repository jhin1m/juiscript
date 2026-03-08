# Phase 2: Site + PHP Commands

## File: `cmd/juiscript/cmd-site.go`

### Command Tree

```
juiscript site list
juiscript site info --domain example.com
juiscript site create --domain example.com --type laravel --php 8.3
juiscript site delete --domain example.com [--remove-db]
juiscript site enable --domain example.com
juiscript site disable --domain example.com
```

### Implementation

```go
func siteCmd(mgrs *Managers) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "site",
        Short: "Manage sites",
    }
    cmd.AddCommand(siteListCmd(mgrs))
    cmd.AddCommand(siteInfoCmd(mgrs))
    cmd.AddCommand(siteCreateCmd(mgrs))
    cmd.AddCommand(siteDeleteCmd(mgrs))
    cmd.AddCommand(siteEnableCmd(mgrs))
    cmd.AddCommand(siteDisableCmd(mgrs))
    return cmd
}
```

### Subcommand Details

**site list**
- Calls `mgrs.Site.List()`
- Output table: DOMAIN | TYPE | PHP | STATUS | CREATED

**site info --domain X**
- Calls `mgrs.Site.Get(domain)`
- Output key-value pairs: Domain, User, Type, PHP, WebRoot, DB, SSL, Status, Created

**site create --domain X --type X --php X [--create-db]**
- Flags: `--domain` (required), `--type` (required, choices: laravel/wordpress), `--php` (default: from config), `--create-db` (bool)
- Calls `mgrs.Site.Create(site.CreateOptions{...})`
- Print: "Site created: example.com"

**site delete --domain X [--remove-db]**
- Flags: `--domain` (required), `--remove-db` (bool, default false)
- Calls `mgrs.Site.Delete(domain, removeDB)`
- Print: "Site deleted: example.com"

**site enable/disable --domain X**
- Calls `mgrs.Site.Enable(domain)` / `mgrs.Site.Disable(domain)`

---

## File: `cmd/juiscript/cmd-php.go`

### Command Tree

```
juiscript php list
juiscript php install --version 8.3
juiscript php remove --version 8.2
```

### Subcommand Details

**php list**
- Calls `mgrs.PHP.ListVersions(ctx)`
- Output table: VERSION | FPM STATUS | ENABLED

**php install --version X**
- Calls `mgrs.PHP.InstallVersion(ctx, version)`
- Print: "PHP 8.3 installed"

**php remove --version X**
- Calls `mgrs.PHP.RemoveVersion(ctx, version, nil)` (pass nil for activeSites -- CLI user is responsible)
- Note: Could warn if sites use version, but YAGNI for v1. TUI already has this protection.
- Print: "PHP 8.2 removed"

### Context

All commands create `context.Background()`. Long-running ops (install/remove) rely on manager's internal timeouts.

## Acceptance Criteria

- [ ] All 6 site subcommands functional
- [ ] All 3 php subcommands functional
- [ ] Flag validation with clear error messages
- [ ] Tabular output for list commands
- [ ] `go build` succeeds
