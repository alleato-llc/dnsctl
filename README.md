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
- **Privileged helper** - Optional root daemon for password-less DNS/hosts changes without `sudo`
- **Desktop GUI** - Optional [Wails](https://wails.io) (Go + TypeScript) frontend on the same core (see [Desktop GUI](#desktop-gui-wails))

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

## Privileged helper (password-less changes)

Changing DNS settings and writing `/etc/hosts` require root. You can either run
the mutating commands with `sudo`, or install the **dnsctl-helper** — a small
root LaunchDaemon that performs the privileged work on your behalf so the CLI
(and a future GUI) can make changes without `sudo` each time.

```bash
make build            # builds bin/dnsctl and bin/dnsctl-helper
make install-helper   # installs + starts the helper (asks for sudo once)
# ...later:
make uninstall-helper
```

How it routes:

- **Running as root** (`sudo dnsctl ...`): the change is performed in-process.
- **Running unprivileged**: the change is forwarded to the helper over a unix
  socket at `/var/run/dnsctl-helper.sock`. If the helper isn't installed, write
  commands report a clear connection error (reads still work).

Security model: the helper is the trust boundary. The socket is world-
connectable, and access is gated by a **peer-UID check** (`LOCAL_PEERCRED`) —
only root and the UID authorized at install time (your user) may drive it. It
restricts hosts writes to `/etc/hosts`, validates DNS server IPs, and re-parses/
validates hosts content before writing. Set `DNSCTL_HELPER_SOCKET` to point the
CLI at a non-default socket.

> Note: for source/Homebrew installs the helper runs unsigned, which is fine
> locally. Distributing a notarized app would additionally require code-signing
> the helper (and, for a bundled GUI, registering it via `SMAppService`).

## Desktop GUI (Wails)

An optional desktop GUI lives in [`gui/`](gui/), built with
[Wails](https://wails.io) (Go backend, TypeScript frontend). It binds
`guiapi.App`, which sits on the same `internal/service` facade as the CLI and
TUI, so all three share one core. The GUI is a **separate Go module** (its own
`go.mod`) so the Wails/CGo dependency tree stays out of the main build.

### Prerequisites

- Go 1.24+ and Node.js + npm
- The Wails v2 CLI — install it, then verify system dependencies with
  `wails doctor`:
  ```bash
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  wails doctor
  ```
  `go install` places the binary in Go's bin directory (`$(go env GOPATH)/bin`,
  typically `~/go/bin`). If `wails doctor` reports `command not found`, that
  directory isn't on your `PATH`. Add it (zsh):
  ```bash
  echo 'export PATH="$PATH:$HOME/go/bin"' >> ~/.zshrc
  source ~/.zshrc
  ```
- The privileged helper, since the GUI runs unprivileged and forwards changes
  to it:
  ```bash
  make build && make install-helper
  ```

### First-time setup

Resolve the Wails dependency tree, then generate the TypeScript bindings for
`guiapi.App` into `frontend/wailsjs/`:

```bash
cd gui
go mod tidy
wails generate module
```

### Run / build

`wails dev` runs a hot-reloading development build; `wails build` produces a
production app at `gui/build/bin/dnsctl-gui.app`:

```bash
cd gui
wails dev
```

```bash
cd gui
wails build
```

See [`gui/README.md`](gui/README.md) for architecture details and how to add
new bound methods.

> Tip: copy commands without the surrounding comments. In an interactive zsh
> shell, `#` is not treated as a comment by default, so a pasted
> `go mod tidy   # ...` runs with the comment as arguments and fails.

## Permissions

Changing DNS settings on macOS requires appropriate permissions. You may need to:

1. Run with `sudo`:
   ```bash
   sudo dnsctl
   ```

2. Or grant your terminal Full Disk Access in **System Preferences > Privacy & Security > Full Disk Access**

3. Or install the [privileged helper](#privileged-helper-password-less-changes)
   for password-less changes without `sudo`.

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
├── cmd/
│   ├── dnsctl/                   # Main binary: CLI + TUI
│   │   ├── main.go               #   Entry point (calls Execute)
│   │   ├── root.go               #   Cobra root command; default action runs the TUI
│   │   ├── runner.go             #   chooseRunner(): in-process if root, else helper
│   │   └── hosts.go              #   `dnsctl hosts` subcommands
│   └── dnsctl-helper/            # Privileged root daemon
│       ├── main.go               #   Serves privileged ops over a unix socket
│       └── peercred_*.go         #   Peer-UID authorization (LOCAL_PEERCRED)
├── internal/
│   ├── config/                  # YAML config loading
│   ├── dns/                     # networksetup wrapper + mock client
│   ├── hosts/                   # /etc/hosts managed-block parser, CRUD, atomic write
│   ├── ipc/                     # Helper request/response protocol
│   ├── service/                 # Privilege-agnostic facade (Hosts/ResolverService)
│   │                            #   + PrivilegedRunner: DirectRunner | HelperClient
│   └── tui/                     # Bubble Tea TUI
├── guiapi/                      # GUI binding layer on the facade (testable, main module)
├── gui/                         # Wails desktop app — SEPARATE go.mod
│   ├── main.go                   #   Wails bootstrap, binds guiapi.App
│   ├── wails.json                #   Wails project config
│   └── frontend/                 #   TypeScript frontend
├── packaging/                   # LaunchDaemon plist + helper install/uninstall scripts
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
