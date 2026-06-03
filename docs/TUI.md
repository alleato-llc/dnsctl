# TUI Reference

The interactive terminal UI, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).
Launch it by running dnsctl with no subcommand:

```bash
dnsctl
```

For the same operations headlessly (scripts, agents), see the [CLI](CLI.md).

## Keybindings

### Main screen

| Key | Action |
|-----|--------|
| `p` | Switch DNS profile |
| `c` | Clear DNS (use DHCP) |
| `s` | Change network service |
| `r` | Refresh status |
| `q` | Quit |

### List views

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` | Select |
| `Esc` | Go back |
| `q` | Quit |

## Layout

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

Applying a profile or clearing DNS changes resolver config, which requires root
— run `sudo dnsctl`, or install the
[privileged helper](DESIGN.md#the-privileged-helper-trust-boundary) for
password-less changes.
