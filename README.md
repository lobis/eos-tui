# EOS TUI

A terminal user interface for monitoring and managing EOS storage clusters.

## Features

- **MGM & QDB Monitoring**: specialized views for Management Nodes and the QuarkDB cluster (Raft quorum).
- **FST View**: monitor storage nodes, disk load, and traffic.
- **Namespace Browser**: browse the EOS file system, inspect file metadata, and view layouts.
- **Space Management**: view space distribution, quotas, and health.
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

## Usage

Start the TUI by specifying an SSH target (gateway) for the EOS cluster:

```bash
eos-tui -ssh eospilot
```

### Command Line Arguments

- `-ssh`: gateway/initial SSH host (e.g. `eospilot`). The tool will automatically discover the MGM leader and route subsequent commands directly.
- `-path`: initial namespace path to browse (default `/`).
- `-timeout`: per-request timeout (default `15s`).
- `-no-alt-screen`: disable alternate screen mode.

## Keybindings

### Global

- `tab` / `1-9`: Switch between views.
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

History of all executed EOS commands and their responses is kept in `~/.eos-tui/history.log`.
