# EOS TUI

A terminal user interface for monitoring and managing [EOS](https://eos-web.web.cern.ch/) storage clusters, inspired by [k9s](https://k9scli.io/).

## Installation

### One-liner (Linux & macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/lobis/eos-tui/main/install.sh | bash
```

Downloads the correct binary for your OS and architecture, verifies the SHA256 checksum, and installs to `/usr/local/bin`. Use `INSTALL_DIR` to override the destination:

```bash
curl -fsSL https://raw.githubusercontent.com/lobis/eos-tui/main/install.sh | INSTALL_DIR=~/.local/bin bash
```

### Homebrew (macOS)

```bash
brew install lobis/tap/eos-tui
```

### RPM (AlmaLinux / RHEL)

Pre-built RPMs for AlmaLinux 9 and 10 (x86\_64 and aarch64) are attached to every [GitHub release](https://github.com/lobis/eos-tui/releases/latest):

```bash
# AlmaLinux 9, x86_64 — replace the version and filename as needed
VERSION=v0.0.3
curl -fsSL "https://github.com/lobis/eos-tui/releases/download/${VERSION}/eos-tui-${VERSION#v}-1.el9.x86_64.rpm" -o eos-tui.rpm
sudo rpm -i eos-tui.rpm
```

### From source

```bash
go install github.com/lobis/eos-tui@latest
```

Requires Go 1.21+. Make sure `$GOPATH/bin` (or `$HOME/go/bin`) is in your `PATH`.

---

## Usage

```bash
eos-tui --ssh <gateway>
```

The tool connects to the cluster via SSH, discovers the MGM leader automatically, and routes all subsequent commands directly to it.

### Examples

```bash
# Connect via an SSH alias defined in ~/.ssh/config
eos-tui --ssh eospilot

# Auto-accept first-seen SSH host keys (useful for new hosts)
eos-tui --ssh eospilot --ssh-accept-new-host-keys

# Run against a local EOS instance (no SSH)
eos-tui
```

### Options

| Flag | Env variable | Default | Description |
|---|---|---|---|
| `--ssh` | `EOS_TUI_SSH` | _(local)_ | SSH gateway / initial target |
| `--ssh-accept-new-host-keys` | `EOS_TUI_SSH_ACCEPT_NEW_HOST_KEYS` | `false` | Accept new host keys automatically |
| `--timeout` | `EOS_TUI_TIMEOUT` | `15s` | Per-request timeout |
| `--no-alt-screen` | `EOS_TUI_NO_ALT_SCREEN` | `false` | Disable alternate screen |
| `--version` | — | — | Print version and exit |

---

## Features

- **MGM & QDB view** — Raft quorum status, leader election, per-node EOS version
- **FST view** — storage nodes, disk load, net traffic
- **Filesystem view** — per-filesystem status, configstatus editing (`rw`/`ro`/`drain`/`empty`)
- **Namespace browser** — browse the EOS namespace, inspect file metadata and layouts
- **Space & group monitoring** — capacity, usage, quotas, health
- **Real-time IO traffic** — live IO shaping by app, user, or group
- **Log viewer** — tail and grep logs from any node, with live updates
- **Integrated shell** — open an interactive SSH shell to any selected node
- **Apollon drain** — trigger drain workflows directly from the filesystem view
- **Recent commands panel** — full history of every command issued this session

## Keybindings

### Global

| Key | Action |
|---|---|
| `tab` / `0`–`9` | Switch view |
| `r` | Refresh current view |
| `l` | Open log overlay for selected node |
| `s` | Open SSH shell to selected node |
| `L` | Toggle recent-commands panel |
| `q` / `ctrl+c` | Quit |

### Navigation & filtering

| Key | Action |
|---|---|
| `↑` / `↓` (or `j` / `k`) | Move selection |
| `←` / `→` | Move column focus |
| `S` | Cycle sort on focused column |
| `/` | Filter current view |
| `esc` | Clear filter / close overlay |
| `enter` | Edit selected value (where applicable) |

---

## Session data

| Path | Contents |
|---|---|
| `~/.eos-tui/sessions/` | Per-session command logs (timestamped) |
| `~/.eos-tui/latest.log` | Symlink to the most recent session log |
| `~/.eos-tui/ui-state.json` | Persisted UI state (active view, namespace path, panel visibility) |

---

## Releasing

Push a tag matching `vMAJOR.MINOR.PATCH` to trigger the release workflow:

```bash
git tag v1.2.3 && git push origin v1.2.3
```

The workflow builds binaries for Linux (amd64/arm64), macOS (arm64), and Windows (amd64), produces RPMs for AlmaLinux 9/10, publishes a GitHub release with `SHA256SUMS.txt`, and updates the Homebrew formula automatically.

Set the `HOMEBREW_TAP_GITHUB_TOKEN` repository secret (with `contents:write` on `lobis/homebrew-tap`) to enable automatic Homebrew formula updates.
