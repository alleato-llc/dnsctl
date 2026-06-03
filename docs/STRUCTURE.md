# Project Structure

How the repository is laid out. For *why* it's shaped this way — the service
facade, the privilege seam, and the separate GUI module — see
[DESIGN.md](DESIGN.md).

```
dnsctl/
├── cmd/
│   ├── dnsctl/                   # Main binary: CLI + TUI
│   │   ├── main.go               #   Entry point (calls Execute)
│   │   ├── root.go               #   Cobra root command; default action runs the TUI
│   │   ├── runner.go             #   chooseRunner(): in-process if root, else helper
│   │   ├── hosts.go              #   `dnsctl hosts` subcommands
│   │   └── profile.go            #   `dnsctl profile` subcommands
│   └── dnsctl-helper/            # Privileged root daemon
│       ├── main.go               #   Serves privileged ops over a unix socket
│       └── peercred_*.go         #   Peer-UID authorization (LOCAL_PEERCRED)
├── internal/
│   ├── config/                  # YAML config loading (+ platform defaults)
│   ├── dns/                     # networksetup wrapper + mock client
│   ├── hosts/                   # /etc/hosts managed-block parser, CRUD, atomic write
│   ├── ipc/                     # Helper request/response protocol
│   ├── service/                 # Privilege-agnostic facade (Hosts/Resolver/ProfileService)
│   │                            #   + PrivilegedRunner: DirectRunner | HelperClient
│   └── tui/                     # Bubble Tea TUI
├── guiapi/                      # GUI binding layer on the facade (testable, main module)
├── gui/                         # Wails desktop app — SEPARATE go.mod
│   ├── main.go                   #   Wails bootstrap, binds guiapi.App
│   ├── wails.json                #   Wails project config
│   └── frontend/                 #   TypeScript frontend
├── packaging/                   # LaunchDaemon plist + helper install/uninstall scripts
├── docs/                        # This documentation
├── go.mod
├── Makefile
└── config.example.yaml
```

## Where things live

| Layer | Package | Notes |
|-------|---------|-------|
| Entry points | `cmd/dnsctl` | Cobra root runs the TUI; subcommands are headless |
| Privileged daemon | `cmd/dnsctl-helper` | The trust boundary (see [DESIGN.md](DESIGN.md)) |
| Shared facade | `internal/service` | `HostsService`, `ResolverService`, `ProfileService` |
| Domain logic | `internal/{config,dns,hosts}` | Pure, system-command-free where possible |
| TUI | `internal/tui` | Bubble Tea model/update/view |
| GUI binding | `guiapi` | Frontend-facing methods; main module, unit-tested |
| GUI app | `gui` | Wails (separate `go.mod`); see [gui/README.md](../gui/README.md) |

The frontend never calls the domain packages directly — the CLI, TUI, and GUI
all sit on `internal/service`, so behavior stays consistent across them.
