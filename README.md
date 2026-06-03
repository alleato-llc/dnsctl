# dnsctl

[![CI](https://github.com/alleato-llc/dnsctl/actions/workflows/ci.yml/badge.svg)](https://github.com/alleato-llc/dnsctl/actions/workflows/ci.yml)
[![GitHub Release](https://img.shields.io/github/v/release/alleato-llc/dnsctl)](https://github.com/alleato-llc/dnsctl/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/alleato-llc/dnsctl)](go.mod)
[![License](https://img.shields.io/github/license/alleato-llc/dnsctl)](LICENSE)
[![Built with Claude](https://img.shields.io/badge/Built%20with-Claude-blueviolet)](https://claude.ai)

Switch between DNS server profiles on macOS — from a terminal UI, headless CLI
subcommands, or a desktop app, all on one shared core.

## Overview

dnsctl wraps the macOS networking tools (`networksetup`, `dscacheutil`) behind a
friendlier interface. Define named **profiles** ("home", "work", "traveling")
and switch your active DNS with a keypress, a one-liner, or a click. It also
manages your `/etc/hosts` entries and can run a small root **helper** so changes
don't need `sudo` each time.

Three frontends, one behavior:

- **TUI** — an interactive terminal UI ([details](docs/TUI.md))
- **CLI** — scriptable `profile` and `hosts` subcommands with JSON output ([details](docs/CLI.md))
- **GUI** — an optional [Wails](https://wails.io) desktop app ([details](gui/README.md))

## Quick start

Install (build from source — see [docs/INSTALL.md](docs/INSTALL.md) for the full
guide, including the helper and GUI):

```bash
git clone https://github.com/alleato-llc/dnsctl.git
cd dnsctl
make build
make install        # to /usr/local/bin (requires sudo)
make config         # seed ~/.config/dnsctl/config.yaml
```

### TUI

```bash
dnsctl
```

`p` switches profile, `c` clears DNS (DHCP), `s` changes service, `q` quits.
Full keybindings and layout in [docs/TUI.md](docs/TUI.md).

### CLI

```bash
dnsctl profile list                                 # list profiles
sudo dnsctl profile apply cloudflare                # apply (resolver write needs root)
dnsctl hosts set myapp.local 127.0.0.1 --dry-run    # preview a hosts edit
```

Reads are unprivileged; writes need root (`sudo`, or the helper). Full command
and flag reference in [docs/CLI.md](docs/CLI.md).

### GUI

A System Settings-style desktop app (DNS Status, Profiles, Hosts, Settings).
It runs unprivileged and forwards changes to the helper, so install that first:

```bash
make build && sudo make install-helper
cd gui && wails build       # -> gui/build/bin/dnsctl-gui.app
```

Prerequisites, `wails generate module`, and building a DMG (`make gui-dmg`) are
covered in [docs/INSTALL.md](docs/INSTALL.md#3-desktop-gui-wails); the app's
architecture in [gui/README.md](gui/README.md).

## Password-less changes (optional)

Changing DNS and writing `/etc/hosts` need root. The CLI can use `sudo`, but the
GUI can't elevate itself — so install the **dnsctl-helper**, a root LaunchDaemon
that performs the privileged work for you after a one-time install:

```bash
make build-helper
sudo make install-helper      # the only time you enter a password
```

How it's authorized and the security tradeoffs are described in
[docs/DESIGN.md](docs/DESIGN.md#the-privileged-helper-trust-boundary);
install/uninstall steps in [docs/INSTALL.md](docs/INSTALL.md).

## Documentation

| Doc | What's in it |
|-----|--------------|
| [docs/INSTALL.md](docs/INSTALL.md) | Full install guide — CLI, helper, GUI, DMG |
| [docs/CLI.md](docs/CLI.md) | `profile` and `hosts` command + flag reference, config format |
| [docs/TUI.md](docs/TUI.md) | TUI keybindings and layout |
| [gui/README.md](gui/README.md) | GUI architecture and bound methods |
| [docs/DESIGN.md](docs/DESIGN.md) | Architecture: the service facade, privilege seam, helper |
| [docs/STRUCTURE.md](docs/STRUCTURE.md) | Repository layout |
| [docs/RELEASE.md](docs/RELEASE.md) | Release process and commit conventions |

## Contributing

This project uses [Conventional Commits](https://www.conventionalcommits.org/)
to drive automated releases — e.g. `feat: add quad9 profile`, `fix: …`,
`docs: …`. Details and the release flow are in [docs/RELEASE.md](docs/RELEASE.md).

## License

MIT
