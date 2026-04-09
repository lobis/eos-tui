# EOS TUI

A terminal user interface for monitoring and managing EOS storage clusters.

## Features

- **MGM & QDB Monitoring**: specialized views for Management Nodes and the QuarkDB cluster (Raft quorum).
- **FST View**: monitor storage nodes, disk load, and traffic.
- **Namespace Browser**: browse the EOS file system, inspect file metadata, and view layouts.
- **Space Management**: view space distribution, quotas, and health.
- **Group Monitoring**: monitor EOS groups, capacity usage, and file counts.
- **Real-time IO Traffic**: live tracking of IO shaping and traffic by app, user, or group.
- **Interactive Logs**: tail and grep logs from any node directly in the TUI.
- **Integrated Shell**: open an interactive SSH shell to any node with a single keypress.

## Installation

### Via Go Install

You can install the latest version directly using `go install`:

```bash
go install github.com/lobis/eos-tui@latest
```

*Note: Ensure `$GOPATH/bin` is in your `PATH`.*

### From Source

Clone the repository and build using the provided Makefile:

```bash
git clone https://github.com/lobis/eos-tui.git
cd eos-tui
make build
```

The binary will be available in the `./bin` directory.

## Releases

Creating and pushing a tag that matches `vMAJOR.MINOR.PATCH`, for example
`v0.1.2`, triggers the GitHub release workflow.

Each release publishes downloadable binaries for:

- macOS amd64
- Windows amd64
- Linux amd64
- Linux arm64

The Linux release binary is built with `CGO_ENABLED=0`, so the same artifact is
intended to work across Ubuntu and AlmaLinux 9/10 without separate distro-specific
builds.

The workflow also attaches a `SHA256SUMS.txt` file to the GitHub Release so the
artifacts can be verified after download.

## Usage

Start the TUI by specifying an SSH target (gateway) for the EOS cluster:

```bash
eos-tui --ssh eospilot
```

If you want EOS TUI to automatically trust first-seen SSH host keys instead of
showing the OpenSSH confirmation prompt, enable:

```bash
eos-tui --ssh eospilot --ssh-accept-new-host-keys
```

### Command Line Arguments

- `--ssh`: gateway/initial SSH host (e.g. `eospilot`). The tool will automatically discover the MGM leader and route subsequent commands directly.
- `--ssh-accept-new-host-keys`: opt in to `StrictHostKeyChecking=accept-new` for SSH connections. This auto-accepts new host keys but still rejects changed keys.
- `--timeout`: per-request timeout (default `15s`).
- `--no-alt-screen`: disable alternate screen mode.

Environment variables:

- `EOS_TUI_SSH`: same as `--ssh`.
- `EOS_TUI_SSH_TARGET`: compatibility alias for `--ssh`.
- `EOS_TUI_SSH_ACCEPT_NEW_HOST_KEYS`: same as `--ssh-accept-new-host-keys`.
- `EOS_TUI_TIMEOUT`: same as `--timeout`.
- `EOS_TUI_NO_ALT_SCREEN`: same as `--no-alt-screen`.

## Keybindings

### Global

- `tab` / `0-9`: Switch between views.
- `r`: Refresh current view.
- `l`: Open log overlay for the selected node.
- `s`: Open an interactive SSH shell to the selected node.
- `q` / `ctrl+c`: Quit.

### Navigation

- `↑/↓` (or `j/k`): Scroll through lists.
- `←/→`: Change selected columns for filtering/sorting.
- `S`: Cycle through sort modes for the selected column.
- `f` (or `/`): Filter the current view.
- `esc`: Clear active filter or close popups.

## Logging

History of all executed EOS commands is kept per session in `~/.eos-tui/sessions/`.
The latest session is also symlinked at `~/.eos-tui/latest.log`.

## Session State

EOS TUI restores lightweight UI state from `~/.eos-tui/ui-state.json`, including:

- the last browsed namespace path
- the last active view
- whether the recent-commands panel was open
