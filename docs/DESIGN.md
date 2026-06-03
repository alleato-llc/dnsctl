# Design

How dnsctl is put together: one shared core behind three frontends, with a
single seam where root privilege is acquired. For the file/directory layout see
[STRUCTURE.md](STRUCTURE.md).

## One core, three frontends

The CLI, TUI, and GUI are all thin shells over `internal/service`, a
privilege-agnostic facade:

- **`HostsService`** — load / validate / mutate / persist `/etc/hosts` entries.
- **`ResolverService`** — read resolver config and perform privileged writes
  (set/clear DNS, flush cache).
- **`ProfileService`** — list / apply / create / edit / delete named DNS
  profiles. Profiles live in the user-owned config, so reads and edits are
  unprivileged; only `Apply` (a resolver write) crosses the privilege seam.

Service types use plain fields with JSON tags so they double as the Wails
(Go → TypeScript) binding surface — the same structs the GUI frontend sees.

```
CLI ─┐
TUI ─┼─► internal/service ─► PrivilegedRunner ─► (root work)
GUI ─┘    (Hosts/Resolver/Profile)
```

## The privilege seam

Operations that need root — changing resolver config, flushing the cache,
writing the hosts file — all funnel through one interface, `PrivilegedRunner`.
There are two implementations:

- **`DirectRunner`** runs the work in-process. Used by `sudo dnsctl` and inside
  the root helper daemon.
- **`HelperClient`** forwards the operation over a unix socket to the privileged
  helper. Used by the unprivileged GUI and a non-root CLI.

`chooseRunner` in `cmd/dnsctl` picks `DirectRunner` when running as root, else
`HelperClient`. To add a privileged operation: put the orchestration in a
`*Service` method, and if it has a root-only side effect, add it to
`PrivilegedRunner` and both implementations.

| How you run it | Mechanism | Helper required? | Password? |
|----------------|-----------|------------------|-----------|
| GUI (always non-root) | forwarded to the helper | **Yes** (writes fail without it) | No, after install |
| `dnsctl …` (non-root) | forwarded to the helper | Yes, for writes | No, after install |
| `sudo dnsctl …` | performed in-process | No | Yes, per `sudo` |

Reads (`hosts list`/`get`, DNS status, profile list) are unprivileged and never
touch the helper.

## The privileged helper (trust boundary)

`cmd/dnsctl-helper` is a separate root daemon, installed as a LaunchDaemon, that
executes privileged operations for unprivileged clients. It is the security
boundary:

- **Authorization.** The socket (`/var/run/dnsctl-helper.sock`) is
  world-connectable; access is gated by a **peer-UID check**
  (`LOCAL_PEERCRED`) — only root and the UID authorized at install time may
  drive it.
- **Validation.** DNS servers must be valid IPs; hosts writes are restricted to
  the `--hosts-file` path, and the content is re-parsed and validated before it
  is written.

`DNSCTL_HELPER_SOCKET` overrides the socket path (used in tests, which exercise
the serve loop / dispatch / authorize gate over a temp socket).

> **Tradeoff:** once the helper is installed, *any* process running as your user
> can change DNS and `/etc/hosts` (via the root helper) **without a password**.
> That is the deliberate cost of password-less convenience. To authenticate per
> change instead, skip the helper and use `sudo dnsctl …` (CLI only).

> For source/Homebrew installs the helper runs unsigned, which is fine locally.
> A notarized distribution would additionally require code-signing the helper
> (and, for a bundled GUI, registering it via `SMAppService`).

See [INSTALL.md](INSTALL.md) for how to install and remove the helper.

## Why the GUI is a separate module

`guiapi` (the binding surface) has no GUI-framework dependency, so it lives in
the **main module** — exported (not `internal/`) so the GUI can import it, and
unit-tested as part of the normal build.

`gui/` is the Wails app with its **own `go.mod`** (`replace … => ../`). Keeping
it a nested module excludes the heavy Wails/CGo dependency tree from the parent's
`go build ./...` and CI, while still letting it import `guiapi`. Architecture and
the bound-method surface are documented in [gui/README.md](../gui/README.md).

## How DNS changes happen (macOS)

Under the hood, dnsctl wraps the standard macOS networking tools:

```bash
networksetup -listallnetworkservices          # List services
networksetup -getdnsservers Wi-Fi             # Get current DNS
networksetup -setdnsservers Wi-Fi 1.1.1.1     # Set DNS
networksetup -setdnsservers Wi-Fi empty       # Clear (use DHCP)
dscacheutil -flushcache                       # Flush DNS cache
```

These live in `internal/dns`, behind a `Client` interface with a mock
implementation (`internal/dns/mock.go`) so the rest of the code — and the tests —
never shells out to the system.
