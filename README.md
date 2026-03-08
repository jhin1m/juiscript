<p align="center">
  <h1 align="center">juiscript</h1>
  <p align="center">
    <strong>LEMP Server Management CLI with TUI</strong><br>
    <em>Quản lý LEMP server qua giao diện dòng lệnh đẹp mắt</em>
  </p>
  <p align="center">
    <a href="https://github.com/jhin1m/juiscript/releases"><img src="https://img.shields.io/github/v/release/jhin1m/juiscript?style=flat-square" alt="Release"></a>
    <a href="https://github.com/jhin1m/juiscript/blob/main/LICENSE"><img src="https://img.shields.io/github/license/jhin1m/juiscript?style=flat-square" alt="License"></a>
    <img src="https://img.shields.io/badge/platform-Ubuntu%2022%20%7C%2024-orange?style=flat-square" alt="Platform">
    <img src="https://img.shields.io/badge/go-1.25+-00ADD8?style=flat-square&logo=go" alt="Go">
  </p>
</p>

---

A single-binary Go tool to manage **Nginx, PHP-FPM, MariaDB, and Redis** on Ubuntu servers — with an interactive TUI built on [Bubble Tea](https://github.com/charmbracelet/bubbletea).

*Công cụ Go đóng gói một file duy nhất, quản lý Nginx, PHP-FPM, MariaDB và Redis trên Ubuntu — có giao diện TUI tương tác.*

## Features

| Feature | Description |
|---------|-------------|
| **Site Management** | Isolated sites (Laravel/WordPress) with per-user Linux accounts |
| **Nginx Vhost** | Auto-generate virtual host configs from templates |
| **Multi PHP** | Multiple PHP versions via `ondrej/php` PPA |
| **SSL** | Let's Encrypt automation via certbot |
| **Database** | MariaDB database & user management |
| **Backup** | Full/partial backup & restore with scheduling |
| **Queue Workers** | Supervisor-managed Laravel queue workers |
| **Service Control** | Start/stop/restart Nginx, PHP-FPM, MariaDB, Redis |

## Quick Start

**One-line install:**

```bash
curl -sSL https://raw.githubusercontent.com/jhin1m/juiscript/main/install.sh | sudo bash
```

```bash
sudo juiscript
```

**Manual install:**

Download from [Releases](https://github.com/jhin1m/juiscript/releases), then:

```bash
sudo install -m 755 juiscript-linux-amd64 /usr/local/bin/juiscript
sudo juiscript
```

## Requirements

- Ubuntu 22.04 / 24.04
- Root access
- Go 1.25+ *(build only / chỉ khi build - ko quan trọng)*

## Build from Source

```bash
make build              # Current platform
make build-linux        # Linux AMD64
make build-linux-arm64  # Linux ARM64
```

## Development

```bash
make dev      # Build & run
make test     # Run tests
make fmt      # Format & vet
make cover    # Coverage report
make clean    # Clean artifacts
```

## Architecture

Single static binary with embedded config templates. TUI powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea), CLI by [Cobra](https://github.com/spf13/cobra).


```
cmd/juiscript/       CLI entry point
internal/
  config/            TOML configuration
  system/            OS commands, file ops, user management
  template/          Embedded templates (Nginx, PHP-FPM, Supervisor)
  tui/               TUI screens, theme, components
  site/              Site lifecycle management
  nginx/             Vhost management
  php/               PHP version & FPM pool management
  database/          MariaDB management
  ssl/               Let's Encrypt automation
  backup/            Backup & restore
  supervisor/        Queue worker management
  service/           systemctl wrapper
  provisioner/       Server provisioning
```

## Config

Config file: `/etc/juiscript/config.toml`

See defaults in [`internal/config/config.go`](internal/config/config.go).

## License

[MIT](LICENSE)
