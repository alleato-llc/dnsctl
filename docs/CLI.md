# CLI Reference

dnsctl's headless subcommands — `profile` and `hosts` — do everything the
interactive [TUI](TUI.md) does, but scriptably, for shell pipelines and agents.
Both support `--json` for machine-readable output.

Running bare `dnsctl` (no subcommand) launches the TUI instead.

## Configuration

Configuration lives at `~/.config/dnsctl/config.yaml` (created from defaults on
first run; `make config` seeds it from the example):

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

Each profile supports:

| Field | Description |
|-------|-------------|
| `description` | Human-readable description shown in the TUI |
| `servers` | List of DNS server IP addresses |
| `dhcp` | Set to `true` to clear DNS and use DHCP (automatic) |

Use `dhcp: true` for profiles where you want the network's own DNS — handy when
traveling or on networks with captive portals.

## `dnsctl profile` — DNS profiles

Map a name to a set of DNS servers (or DHCP), then apply it to a network
service. Profile definitions live in the user config above.

```bash
dnsctl profile list                                 # list configured profiles
dnsctl profile add cloudflare --server 1.1.1.1 --server 1.0.0.1 --description "Cloudflare"
dnsctl profile add travel --dhcp --description "Use the network's DNS"
dnsctl profile apply cloudflare                     # apply to the default/active service
dnsctl profile apply cloudflare --service Ethernet  # apply to a specific service
dnsctl profile rm travel                            # delete a profile definition
```

`list`, `add`, and `rm` only touch the user-owned config file and need no
privileges. `apply` changes resolver config, which **requires root** — run it
with `sudo` or install the [privileged helper](DESIGN.md#the-privileged-helper-trust-boundary).
With no `--service`, `apply` targets the config's `default_service`, then the
active (default-route) service.

| Flag | Commands | Description |
|------|----------|-------------|
| `--json` | all | Emit JSON instead of a table |
| `--config PATH` | all | Operate on a different config file |
| `--service NAME` | `apply` | Network service to apply to (default: config default, then active) |
| `--server IP` | `add` | DNS server (repeatable) |
| `--dhcp` | `add` | Profile reverts the service to DHCP-provided DNS |
| `--description TEXT` | `add` | Human-readable description |

## `dnsctl hosts` — `/etc/hosts` entries

Manage local hostname mappings (e.g. pointing `myapp.local` at `127.0.0.1`).

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

Writing `/etc/hosts` requires root, so run the mutating commands with `sudo`
(or install the helper):

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
