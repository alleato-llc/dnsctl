# dnsctl

[![CI](https://github.com/nycjv321/dnsctl/actions/workflows/ci.yml/badge.svg)](https://github.com/nycjv321/dnsctl/actions/workflows/ci.yml)
[![GitHub Release](https://img.shields.io/github/v/release/nycjv321/dnsctl)](https://github.com/nycjv321/dnsctl/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/nycjv321/dnsctl)](go.mod)
[![License](https://img.shields.io/github/license/nycjv321/dnsctl)](LICENSE)
[![Built with Claude](https://img.shields.io/badge/Built%20with-Claude-blueviolet)](https://claude.ai)

A Unix CLI tool with a TUI for switching between DNS server profiles.

## Features

- **Named DNS profiles** - Define profiles like "home", "traveling", "work" with different DNS servers
- **TUI interface** - ncurses-style terminal UI using Bubble Tea
- **Clear DNS** - Revert to DHCP defaults with a single keypress
- **Multiple network services** - Switch between Wi-Fi, Ethernet, and other interfaces
- **DNS cache flushing** - Automatically flush DNS cache after changes
- **`/etc/hosts` management** - Headless `dnsctl hosts` subcommands to list/add/update/remove local hostname mappings, with JSON output for scripts and agents

## Installation

### Build from source

```bash
# Clone the repository
git clone https://github.com/nycjv321/dnsctl.git
cd dnsctl

# Build
make build

# Install to /usr/local/bin (requires sudo)
make install
```

### Quick start

```bash
# Create config from example
make config

# Run the tool
make run
# or
./bin/dnsctl
```

## Configuration

Configuration is stored at `~/.config/dnsctl/config.yaml`:

```yaml
version: 1
default_service: "Wi-Fi"

profiles:
  home:
    description: "Home network with Pi-hole"
    servers: ["192.168.1.100", "1.1.1.1"]
  traveling:
    description: "Use network's DNS (DHCP)"
    dhcp: true  # Clears DNS settings to use DHCP
  cloudflare:
    description: "Cloudflare DNS"
    servers: ["1.1.1.1", "1.0.0.1"]
  google:
    description: "Google Public DNS"
    servers: ["8.8.8.8", "8.8.4.4"]

settings:
  flush_cache: true
```

### Profile Options

Each profile supports these fields:

| Field | Description |
|-------|-------------|
| `description` | Human-readable description shown in the TUI |
| `servers` | List of DNS server IP addresses |
| `dhcp` | Set to `true` to clear DNS and use DHCP (automatic) |

Use `dhcp: true` for profiles where you want to use the network's default DNS (useful when traveling or on networks with captive portals).

## Usage

Launch the TUI:

```bash
dnsctl
```

### Keybindings

#### Main Screen

| Key | Action |
|-----|--------|
| `p` | Switch DNS profile |
| `c` | Clear DNS (use DHCP) |
| `s` | Change network service |
| `r` | Refresh status |
| `q` | Quit |

#### List Views

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` | Select |
| `Esc` | Go back |
| `q` | Quit |

### TUI Layout

```
Main Screen                    Profile Selection
┌─────────────────────┐       ┌─────────────────────┐
│ Current: Wi-Fi      │  [p]  │ > home              │
│ DNS: 1.1.1.1        │ ───── │   traveling         │
│                     │       │   cloudflare        │
│ [p] Switch Profile  │       │                     │
│ [c] Clear DNS       │       │ Servers: 192.168... │
│ [s] Change Service  │       └─────────────────────┘
│ [q] Quit            │
└─────────────────────┘
```

## Managing `/etc/hosts` entries

The `dnsctl hosts` subcommand manages local hostname mappings (e.g. pointing
`myapp.local` at `127.0.0.1`) from the command line — handy for scripts and
LLM/agent invocation where the interactive TUI isn't appropriate.

dnsctl only ever edits its own **managed block**, delimited by sentinel
comments. Everything outside that block — `localhost`, `broadcasthost`, and any
lines you added by hand — is preserved exactly:

```
# BEGIN dnsctl (managed by `dnsctl hosts` — do not edit by hand)
127.0.0.1	myapp.local	# dev box
10.0.0.5	staging.api api2.local
# END dnsctl
```

### Commands

```bash
dnsctl hosts list                                   # list managed entries
dnsctl hosts get myapp.local                        # show one entry
dnsctl hosts add myapp.local 127.0.0.1              # add (fails if it exists)
dnsctl hosts set myapp.local 127.0.0.1              # add or update (idempotent)
dnsctl hosts rm  myapp.local                        # remove an entry
```

Writing `/etc/hosts` requires root, so run the mutating commands with `sudo`:

```bash
sudo dnsctl hosts add staging.api 10.0.0.5 --alias api2.local --comment "staging"
```

### Flags

| Flag | Commands | Description |
|------|----------|-------------|
| `--json` | `list`, `get` | Emit JSON instead of a table |
| `--file PATH` | all | Operate on a different file (default `/etc/hosts`) |
| `--alias NAME` | `add`, `set` | Additional hostname for the same IP (repeatable) |
| `--comment TEXT` | `add`, `set` | Trailing comment for the entry |
| `--dry-run` | `add`, `set`, `rm` | Print the resulting file without writing |
| `--flush` | `add`, `set`, `rm` | Flush the DNS cache after writing |

Reads (`list`/`get`) don't need elevated privileges; only writes do. Before each
write the current file is backed up to `<path>.dnsctl.bak`, and the new content
is written atomically (temp file + rename) so an interrupted run can't corrupt
`/etc/hosts`.

## Permissions

Changing DNS settings on macOS requires appropriate permissions. You may need to:

1. Run with `sudo`:
   ```bash
   sudo dnsctl
   ```

2. Or grant your terminal Full Disk Access in **System Preferences > Privacy & Security > Full Disk Access**

## How It Works

dnsctl uses macOS `networksetup` commands under the hood:

```bash
networksetup -listallnetworkservices          # List services
networksetup -getdnsservers Wi-Fi             # Get current DNS
networksetup -setdnsservers Wi-Fi 1.1.1.1     # Set DNS
networksetup -setdnsservers Wi-Fi empty       # Clear (use DHCP)
dscacheutil -flushcache                       # Flush DNS cache
```

## Testing

Run the test suite:

```bash
make test              # Run all tests
go test -v ./...       # Verbose output
go test -cover ./...   # With coverage report

# View coverage in browser
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

The project has comprehensive test coverage:
- **Config tests** - Config loading, parsing, defaults, and profile helpers
- **TUI tests** - Model initialization, message handling, key navigation, DNS operations
- **View tests** - Rendering output for all views

Tests use a mock DNS client (`internal/dns/mock.go`) to avoid requiring system access.

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Cobra](https://github.com/spf13/cobra) - CLI commands and flag parsing
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML parsing

## Project Structure

```
dnsctl/
├── cmd/dnsctl/
│   ├── main.go                  # Entry point (calls Execute)
│   ├── root.go                  # Cobra root command; default action runs the TUI
│   └── hosts.go                 # `dnsctl hosts` subcommands
├── internal/
│   ├── config/
│   │   ├── config.go            # YAML config loading
│   │   └── config_test.go       # Config tests
│   ├── dns/
│   │   ├── client.go            # DNS client interface
│   │   ├── macos.go             # networksetup wrapper
│   │   └── mock.go              # Mock client for testing
│   ├── hosts/
│   │   ├── hosts.go             # Entry type, managed-block parser, CRUD
│   │   ├── store.go             # Atomic file read/write + backup
│   │   └── hosts_test.go        # Parser, CRUD, and store tests
│   └── tui/
│       ├── app.go               # Bubble Tea model
│       ├── app_test.go          # TUI logic tests
│       ├── keys.go              # Keybindings
│       ├── styles.go            # Lip Gloss styling
│       ├── views.go             # View rendering
│       └── views_test.go        # View rendering tests
├── go.mod
├── Makefile
└── config.example.yaml
```

## Contributing

This project uses [Conventional Commits](https://www.conventionalcommits.org/) for commit messages, enabling automated releases and changelog generation.

### Commit Format

```
<type>: <description>
```

### Common Types

| Type | Description | Example |
|------|-------------|---------|
| `feat` | New feature | `feat: add quad9 DNS profile` |
| `fix` | Bug fix | `fix: resolve DNS cache flush on Linux` |
| `docs` | Documentation | `docs: update installation instructions` |
| `refactor` | Code refactoring | `refactor: simplify profile loading` |
| `test` | Adding tests | `test: add TUI navigation tests` |
| `chore` | Maintenance | `chore: update dependencies` |

### Breaking Changes

Add `!` after the type for breaking changes:
```bash
feat!: change config file format
```

## License

MIT
