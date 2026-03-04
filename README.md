# juiscript

Go CLI tool for LEMP server management with a beautiful TUI. Manage Nginx, PHP-FPM, MariaDB, and Redis on Ubuntu 22/24.

## Features

- **Site Management** - Create isolated sites (Laravel/WordPress) with per-user Linux accounts
- **Nginx Vhost** - Generate and manage virtual host configs with templates
- **Multi PHP** - Support multiple PHP versions via ondrej/php PPA
- **SSL** - Let's Encrypt certificate automation via certbot
- **Database** - MariaDB database and user management
- **Backup** - Full/partial backup and restore with scheduling
- **Queue Workers** - Supervisor-managed Laravel queue workers
- **Service Control** - Start/stop/restart Nginx, PHP-FPM, MariaDB, Redis

## Install

One-line install on your Ubuntu server:

```bash
curl -sSL https://raw.githubusercontent.com/jhin1m/juiscript/main/install.sh | sudo bash
```

Then run:

```bash
sudo juiscript
```

### Manual Install

Download the binary from [Releases](https://github.com/jhin1m/juiscript/releases), then:

```bash
sudo install -m 755 juiscript-linux-amd64 /usr/local/bin/juiscript
sudo juiscript
```

## Build from Source

```bash
make build              # Current platform
make build-linux        # Linux AMD64
make build-linux-arm64  # Linux ARM64
```

## Requirements

- Ubuntu 22.04 or 24.04
- Root access
- Go 1.22+ (for building)

## Development

```bash
make dev      # Build and run
make test     # Run tests
make fmt      # Format and vet
make cover    # Coverage report
```

## Architecture

Single binary with embedded config templates. TUI built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

```
cmd/juiscript/   CLI entry point (Cobra)
internal/
  config/        TOML configuration
  system/        OS command execution, file ops, user management
  template/      Embedded config templates (Nginx, PHP-FPM, Supervisor)
  tui/           Bubble Tea screens, theme, components
  site/          Site lifecycle management
  nginx/         Vhost management
  php/           PHP version & FPM pool management
  database/      MariaDB management
  ssl/           Let's Encrypt automation
  backup/        Backup & restore
  supervisor/    Queue worker management
  service/       systemctl wrapper
```

## Config

Config stored at `/etc/juiscript/config.toml`. See defaults in `internal/config/config.go`.

## License

MIT
