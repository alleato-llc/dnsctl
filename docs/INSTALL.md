# Installing dnsctl

dnsctl ships as three pieces, installed independently depending on what you need:

| Piece | What it is | When you need it |
|-------|-----------|------------------|
| `dnsctl` | The CLI + TUI binary | Always |
| `dnsctl-helper` | Privileged root LaunchDaemon | Password-less changes, and **required for the GUI** |
| `dnsctl-gui.app` | The Wails desktop app | Only if you want the GUI |

> **zsh tip:** copy commands without trailing comments. In an interactive zsh
> shell `#` is not a comment, so a pasted `make build   # ...` runs the comment
> as arguments and fails.

## 1. CLI / TUI

```bash
git clone https://github.com/nycjv321/dnsctl.git
cd dnsctl
make build
```
```bash
sudo make install        # copies bin/dnsctl to /usr/local/bin
```

Create a starter config and run:

```bash
make config              # writes ~/.config/dnsctl/config.yaml if missing
dnsctl                   # launches the TUI
```

Reads (`dnsctl hosts list`, `dnsctl profile list`, DNS status) need no
privileges. Writes (`dnsctl profile apply`, `dnsctl hosts add/set/rm`) change
system state and require root — run them with `sudo`, or install the helper
below for password-less operation.

## 2. Privileged helper (optional for the CLI, required for the GUI)

The helper is a small root daemon that performs the privileged work, so the
unprivileged GUI (and a non-root CLI) can apply changes without `sudo` each
time. You enter your password once, at install.

```bash
make build-helper
```
```bash
sudo make install-helper
```

This installs a LaunchDaemon (see [`packaging/`](../packaging/)) and authorizes
the installing user. Remove it with:

```bash
sudo make uninstall-helper
```

See the [design notes](DESIGN.md#the-privileged-helper-trust-boundary) for how
the helper authorizes callers (peer-UID over a unix socket) and re-validates
every request.

## 3. Desktop GUI (Wails)

The GUI runs **unprivileged** and forwards every DNS/profile/hosts *write* to
`dnsctl-helper` — so **install the helper (step 2) first**, or writes will fail
(Settings → Helper will show "Not reachable").

### Prerequisites

- Go 1.24+ and Node.js + npm
- The Wails v2 CLI:
  ```bash
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  wails doctor
  ```
  If `wails` is "command not found", add Go's bin dir to `PATH` (zsh):
  ```bash
  echo 'export PATH="$PATH:$HOME/go/bin"' >> ~/.zshrc
  source ~/.zshrc
  ```
- For DMG packaging with the drag-to-Applications layout (optional):
  ```bash
  brew install create-dmg
  ```

### First-time setup

Resolve the Wails dependency tree and generate the TypeScript bindings:

```bash
cd gui
go mod tidy
wails generate module
```

Run a hot-reloading dev build to verify it works:

```bash
cd gui
wails dev
```

### Build a `.app`

From the repo root:

```bash
make gui-build           # -> gui/build/bin/dnsctl-gui.app
```

(Equivalent to `cd gui && wails build`. Add `-platform darwin/universal` to the
underlying `wails build` for a universal Intel+Apple-silicon binary.)

### Build a DMG

```bash
make gui-dmg             # builds the .app, then packages gui/build/bin/dnsctl.dmg
```

`gui-dmg` uses `create-dmg` (a nice window with an Applications drop-link) when
it's installed, and otherwise falls back to the built-in `hdiutil`. The
equivalent manual commands:

```bash
hdiutil create -volname "dnsctl" -srcfolder gui/build/bin/dnsctl-gui.app -ov -format UDZO gui/build/bin/dnsctl.dmg
```
```bash
create-dmg --volname "dnsctl" --app-drop-link 450 120 gui/build/bin/dnsctl.dmg gui/build/bin/dnsctl-gui.app
```

### Install the app locally

Open `gui/build/bin/dnsctl.dmg` and drag the app to Applications, or skip the
DMG entirely:

```bash
cp -R gui/build/bin/dnsctl-gui.app /Applications/
```

### Caveats

- **The helper is not in the DMG.** The DMG/`.app` contains only the GUI;
  install `dnsctl-helper` separately (step 2). Bundling it for true one-click
  distribution would need `SMAppService` registration plus code-signing.
- **Gatekeeper.** A DMG you build and mount *locally* is not quarantined and
  launches normally. But if the DMG is **downloaded or AirDropped** to another
  Mac, the app is unsigned and Gatekeeper will block it. Open it via right-click
  → Open, or clear the quarantine flag:
  ```bash
  xattr -dr com.apple.quarantine /Applications/dnsctl-gui.app
  ```
