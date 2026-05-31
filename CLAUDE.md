# CLAUDE.md

This file provides context for Claude Code when working on this project.

## Project Overview

**dnsctl** is a macOS CLI tool with a TUI for switching between DNS server profiles. It wraps macOS `networksetup` commands with a user-friendly Bubble Tea interface.

## Tech Stack

- **Language**: Go 1.21+
- **TUI Framework**: Bubble Tea (github.com/charmbracelet/bubbletea)
- **Styling**: Lip Gloss (github.com/charmbracelet/lipgloss)
- **Config**: YAML via gopkg.in/yaml.v3
- **Platform**: macOS only (uses `networksetup` and `dscacheutil`)

## Architecture

```
cmd/dnsctl/
├── main.go              # Entry point - calls Execute()
├── root.go              # Cobra root command; default (no subcommand) runs the TUI
└── hosts.go             # `dnsctl hosts` subcommands (list/get/add/set/rm)
internal/
├── config/
│   ├── config.go        # Config loading from ~/.config/dnsctl/config.yaml
│   ├── config_darwin.go # macOS-specific defaults
│   ├── config_linux.go  # Linux-specific defaults
│   └── config_test.go   # Config tests
├── dns/
│   ├── client.go        # Client interface for DNS operations
│   ├── macos.go         # DNS operations wrapper around networksetup
│   └── mock.go          # Mock client for testing
├── hosts/
│   ├── hosts.go         # Entry type, managed-block parser, CRUD, validation
│   ├── store.go         # Atomic file read/write + backup (WriteAtomic primitive)
│   └── hosts_test.go    # Parser, CRUD, and store tests
├── service/            # Privilege-agnostic facade shared by CLI/TUI/GUI
│   ├── hosts.go         # HostsService: orchestrates load/validate/mutate/persist
│   ├── privilege.go     # PrivilegedRunner interface + DirectRunner (in-process)
│   ├── helper_client.go # HelperClient stub (forwards to root helper, TODO)
│   └── hosts_test.go    # Service tests with a recording runner
└── tui/
    ├── app.go           # Bubble Tea Model with Init/Update/View
    ├── app_test.go      # TUI logic tests
    ├── keys.go          # KeyMap for all keybindings
    ├── styles.go        # Lip Gloss style definitions
    ├── views.go         # View rendering functions and View enum
    └── views_test.go    # View rendering tests
```

### Entry point / command layer

The binary uses [Cobra](https://github.com/spf13/cobra). The root command's
`RunE` launches the Bubble Tea TUI, so bare `dnsctl` behaves as it always has.
Subcommands (currently `hosts`) provide headless, scriptable/LLM-friendly
operations. New subcommands are added as files under `cmd/dnsctl/` that register
themselves on `rootCmd` in an `init()`.

### Service facade and the privilege seam

`internal/service` is the shared, privilege-agnostic layer that CLI, TUI, and a
future GUI all build on — frontends should call it rather than the domain
packages directly. Its exported types use plain fields + JSON tags so they can
double as the Wails (Go → TypeScript) binding surface.

Operations that need root (changing resolver config, flushing the cache,
writing the hosts file) go through the `PrivilegedRunner` interface — the single
seam where privilege is acquired:

- `DirectRunner` runs in-process; used by `sudo dnsctl` and (later) inside the
  root helper daemon.
- `HelperClient` (stub today) will forward operations over IPC to a privileged
  helper at `cmd/dnsctl-helper`, for the unprivileged GUI and a non-root CLI.

To add an operation: put orchestration in a `*Service` method, and if it has a
root-only side effect, add it to `PrivilegedRunner` (and both implementations).

## Key Patterns

### Bubble Tea Model

The TUI follows the Elm architecture:
- `Model` struct holds all state (current view, selected index, DNS status, etc.)
- `Init()` returns initial command to fetch DNS status
- `Update()` handles messages and returns new model + commands
- `View()` renders current state to string

### View State Machine

```go
type View int
const (
    ViewMain View = iota      // Main dashboard
    ViewProfiles              // Profile selection list
    ViewServices              // Network service selection list
)
```

### Profile Configuration

Profiles are defined in `internal/config/config.go`:

```go
type Profile struct {
    Description string   `yaml:"description"`
    Servers     []string `yaml:"servers,omitempty"`
    DHCP        bool     `yaml:"dhcp,omitempty"`
}
```

- `IsDHCP()` returns true if `DHCP` is set or `Servers` is empty
- DHCP profiles clear DNS settings instead of setting specific servers
- Useful for "traveling" profiles where you want to use the network's DNS

### DNS Operations

All DNS operations are in `internal/dns/macos.go`:
- `ListNetworkServices()` - Get available network interfaces
- `GetDNSServers(service)` - Get current DNS for a service
- `SetDNSServers(service, servers)` - Set DNS servers
- `ClearDNSServers(service)` - Clear DNS (revert to DHCP)
- `FlushCache()` - Flush DNS cache

### /etc/hosts Management

`internal/hosts/` manages local hostname mappings, surfaced via `dnsctl hosts`.

- **Managed block**: dnsctl only edits lines inside a sentinel-delimited block
  (`# BEGIN dnsctl` ... `# END dnsctl`). Content outside the block is preserved
  byte-for-byte, so system entries are never touched. The block is omitted
  entirely when there are no managed entries.
- `hosts.go` is pure logic over file content: `Parse([]byte) *Document`, then
  `List`/`Get`/`Set`/`Remove` (keyed on hostname, case-insensitive) and
  `Render() []byte`. `Entry.Validate()` checks IP/hostname/alias syntax.
- `store.go` wraps a file path: `Load()` (missing file = empty), and `Save()`
  which backs up to `<path>.dnsctl.bak` then writes atomically (temp file +
  rename) preserving the original mode.
- Reads work unprivileged; writes need root (`sudo`). Tests use `t.TempDir()`
  with a `--file` override rather than touching the real `/etc/hosts`.

## Build Commands

```bash
make build      # Build to bin/dnsctl
make run        # Build and run
make install    # Install to /usr/local/bin (needs sudo)
make config     # Create config from example
make clean      # Remove build artifacts
make test       # Run tests
make fmt        # Format code
```

## Config Location

User config: `~/.config/dnsctl/config.yaml`

Default config is generated if file doesn't exist (see `config.DefaultConfig()`).

## Common Tasks

### Adding a new keybinding

1. Add to `KeyMap` struct in `internal/tui/keys.go`
2. Add to `DefaultKeyMap()` function
3. Handle in appropriate `handle*Keys()` function in `internal/tui/app.go`
4. Update help text in `internal/tui/views.go`

### Adding a new view

1. Add constant to `View` enum in `internal/tui/views.go`
2. Create `render*View()` function in `internal/tui/views.go`
3. Add case to `View()` method in `internal/tui/app.go`
4. Create `handle*Keys()` function in `internal/tui/app.go`
5. Add case to `handleKeyPress()` in `internal/tui/app.go`

### Adding a new DNS operation

1. Add method to `Client` in `internal/dns/macos.go`
2. Create command function in `internal/tui/app.go` that returns `tea.Cmd`
3. Handle result message in `Update()`

### Adding a new CLI subcommand

1. Create `cmd/dnsctl/<name>.go` with a `*cobra.Command`
2. Register it on `rootCmd` (or a parent command) from an `init()`
3. Keep business logic in an `internal/` package; the command file should only
   wire flags/args to that package and format output

## Testing

### Running Tests

```bash
make test              # Run all tests
go test -v ./...       # Verbose output
go test -cover ./...   # With coverage report
go test -v ./internal/tui/...   # Run specific package

# View coverage in browser (opens interactive HTML report)
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

### Test Architecture

The project uses a mock DNS client for testing, avoiding system command dependencies:

```go
// internal/dns/mock.go
type MockClient struct {
    Services   []string              // Configurable service list
    DNSServers map[string][]string   // DNS servers by service
    SetError   error                 // Inject errors for testing
    ClearError error
    FlushError error
    SetCalls   []SetDNSCall          // Record calls for assertions
    ClearCalls []string
    FlushCalls int
}
```

### Test Files

| File | Coverage | Description |
|------|----------|-------------|
| `internal/config/config_test.go` | ~85% | Config loading, parsing, defaults, profile helpers |
| `internal/tui/app_test.go` | ~89% | Model init, Update(), key handlers, DNS operations |
| `internal/tui/views_test.go` | ~89% | View rendering, helper functions |

### Key Test Patterns

**Testing Update() with messages:**
```go
func TestUpdate_StatusMsg_Success(t *testing.T) {
    model, _ := testModel()
    msg := statusMsg{services: []string{"Wi-Fi"}, dnsServers: []string{"1.1.1.1"}}
    newModel, cmd := model.Update(msg)
    m := newModel.(Model)
    // Assert on m.services, m.currentDNS, etc.
}
```

**Testing key navigation:**
```go
func TestProfilesView_NavigateDown(t *testing.T) {
    model, _ := testModel()
    model.currentView = ViewProfiles
    msg := tea.KeyMsg{Type: tea.KeyDown}
    newModel, _ := model.Update(msg)
    // Assert selectedIndex changed
}
```

**Testing DNS operations with mock:**
```go
func TestApplyProfile_SetsDNS(t *testing.T) {
    model, mock := testModel()
    profile := config.Profile{Servers: []string{"9.9.9.9"}}
    cmd := model.applyProfile("test", profile)
    cmd()  // Execute the command
    // Assert mock.SetCalls contains expected call
}
```

### Writing New Tests

1. Use `testModel()` helper from `app_test.go` for TUI tests
2. Use `testConfig()` helper for consistent test configuration
3. Tests are in the same package (`tui`, `config`) for access to unexported fields
4. Use `t.TempDir()` for file-based config tests

## Conventional Commits

This project uses [Conventional Commits](https://www.conventionalcommits.org/) for commit messages. This enables automated versioning and changelog generation via release-please.

### Commit Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Commit Types

| Type | Description | Version Bump |
|------|-------------|--------------|
| `feat` | New feature | Minor (0.x.0) |
| `fix` | Bug fix | Patch (0.0.x) |
| `docs` | Documentation only | None |
| `style` | Formatting, no code change | None |
| `refactor` | Code change that neither fixes nor adds | None |
| `perf` | Performance improvement | Patch (0.0.x) |
| `test` | Adding/updating tests | None |
| `chore` | Maintenance tasks | None |
| `ci` | CI/CD changes | None |

### Breaking Changes

For breaking changes, either:
- Add `!` after the type: `feat!: change config format`
- Add `BREAKING CHANGE:` in the commit body

Breaking changes trigger a major version bump (x.0.0).

### Examples

```bash
# Feature
git commit -m "feat: add quad9 DNS profile"

# Bug fix
git commit -m "fix: resolve DNS cache flush on Linux"

# Breaking change
git commit -m "feat!: change config file format"

# With scope
git commit -m "feat(tui): add keyboard shortcut for flush"

# With body
git commit -m "fix: handle empty DNS response

The DNS client now returns an empty slice instead of nil
when no DNS servers are configured."
```

## Permissions Note

Changing DNS requires elevated privileges. Users may need to run with `sudo` or grant terminal Full Disk Access.
