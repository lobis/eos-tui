package ui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lobis/eos-tui/eos"
)

const commandLogRefreshInterval = 300 * time.Millisecond
const logRefreshInterval = 500 * time.Millisecond
const commandLogTailLines = 200
const startupSplashTickInterval = 120 * time.Millisecond
const apollonCommandTimeout = 30 * time.Second

// checkEOSCmd verifies that `eos version` succeeds (locally or via SSH).
// Must be the first command fired from Init so a helpful fatal popup is shown
// before any other work starts if EOS is unreachable.
func checkEOSCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return eosCheckResultMsg{}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, err := client.EOSVersion(ctx)
		return eosCheckResultMsg{err: err}
	}
}

// loadInfraCmd fans out all infrastructure fetches in parallel.  Each
// component delivers its own typed message to the Bubble Tea runtime as soon
// as it completes, so a slow or timing-out command (e.g. NodeStats) never
// delays the display of faster data (e.g. FST node list).
func loadInfraCmd(c *eos.Client) tea.Cmd {
	return tea.Batch(
		loadNodeStatsCmd(c),
		loadFSTsCmd(c),
		loadMGMsCmd(c),
		loadFileSystemsCmd(c),
		loadEOSVersionCmd(c),
		loadSpacesCmd(c),
		loadNamespaceStatsCmd(c),
	)
}

func loadNodeStatsCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		stats, err := client.NodeStats(context.Background())
		return nodeStatsLoadedMsg{stats: stats, err: err}
	}
}

func loadFSTsCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		fsts, err := client.Nodes(context.Background())
		return fstsLoadedMsg{fsts: fsts, err: err}
	}
}

func loadMGMsCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		mgms, err := client.MGMs(context.Background())
		return mgmsLoadedMsg{mgms: mgms, err: err}
	}
}

func loadEOSVersionCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		version, _ := client.EOSVersion(context.Background())
		return eosVersionLoadedMsg{version: version}
	}
}

func loadFileSystemsCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		fileSystems, err := client.FileSystems(context.Background())
		return fileSystemsLoadedMsg{fs: fileSystems, err: err}
	}
}

func loadSpacesCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		spaces, err := client.Spaces(context.Background())
		return spacesLoadedMsg{spaces: spaces, err: err}
	}
}

func loadGroupsCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		groups, err := client.Groups(context.Background())
		return groupsLoadedMsg{groups: groups, err: err}
	}
}

func loadNamespaceStatsCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		stats, err := client.NamespaceStats(context.Background())
		return namespaceStatsLoadedMsg{stats: stats, err: err}
	}
}

func loadDirectoryCmd(client *eos.Client, dirPath string) tea.Cmd {
	return func() tea.Msg {
		directory, err := client.ListPath(context.Background(), dirPath)
		return directoryLoadedMsg{directory: directory, err: err}
	}
}

func loadNamespaceAttrsCmd(client *eos.Client, path string) tea.Cmd {
	return func() tea.Msg {
		attrs, err := client.ListAttrs(context.Background(), path)
		return namespaceAttrsLoadedMsg{path: path, attrs: attrs, err: err}
	}
}

func runNamespaceAttrSetCmd(client *eos.Client, path, key, value string) tea.Cmd {
	return func() tea.Msg {
		err := client.SetAttr(context.Background(), path, key, value)
		return namespaceAttrSetResultMsg{path: path, err: err}
	}
}

func loadSpaceStatusCmd(client *eos.Client, space string) tea.Cmd {
	return func() tea.Msg {
		records, err := client.SpaceStatus(context.Background(), space)
		return spaceStatusLoadedMsg{space: space, records: records, err: err}
	}
}

func loadIOShapingCmd(client *eos.Client, mode eos.IOShapingMode) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		records, err := client.IOShaping(ctx, mode)
		return ioShapingLoadedMsg{records: records, mode: mode, err: err}
	}
}

func loadIOShapingPoliciesCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		records, err := client.IOShapingPolicies(ctx)
		return ioShapingPoliciesLoadedMsg{records: records, err: err}
	}
}

func runIOShapingPolicySetCmd(client *eos.Client, update eos.IOShapingPolicyUpdate) tea.Cmd {
	return func() tea.Msg {
		err := client.SetIOShapingPolicy(context.Background(), update)
		return ioShapingPolicyResultMsg{id: update.ID, op: "updated", err: err}
	}
}

func runIOShapingPolicyRemoveCmd(client *eos.Client, mode eos.IOShapingMode, id string) tea.Cmd {
	return func() tea.Msg {
		err := client.RemoveIOShapingPolicy(context.Background(), mode, id)
		return ioShapingPolicyResultMsg{id: id, op: "deleted", err: err}
	}
}

func runSpaceConfigCmd(client *eos.Client, space, key, value string) tea.Cmd {
	return func() tea.Msg {
		err := client.SpaceConfig(context.Background(), space, key, value)
		return spaceConfigResultMsg{space: space, err: err}
	}
}

func runGroupSetCmd(client *eos.Client, group, status string) tea.Cmd {
	return func() tea.Msg {
		err := client.SetGroupStatus(context.Background(), group, status)
		return groupSetResultMsg{group: group, status: status, err: err}
	}
}

func runFsConfigStatusCmd(client *eos.Client, fsID uint64, value string) tea.Cmd {
	return func() tea.Msg {
		err := client.FsConfigStatus(context.Background(), fsID, value)
		return fsConfigStatusResultMsg{err: err}
	}
}

func runApollonDrainCmd(client *eos.Client, fsID uint64, instance string) tea.Cmd {
	return func() tea.Msg {
		args := apollonDrainSSHArgs(fsID, instance)
		// Log using flat individual tokens so shellDisplayJoin renders them
		// cleanly, matching the style of every other command in the panel.
		if client != nil {
			logArgs := append([]string{"ssh", apollonSSHTarget}, apollonDrainRemoteArgs(fsID, instance)...)
			client.LogCommand(logArgs)
		}

		ctx, cancel := context.WithTimeout(context.Background(), apollonCommandTimeout)
		defer cancel()

		out, err := exec.CommandContext(ctx, "ssh", args...).CombinedOutput()
		return apollonDrainResultMsg{
			fsID:     fsID,
			instance: instance,
			output:   strings.TrimSpace(string(out)),
			err:      err,
		}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func ioShapingTickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return ioShapingTickMsg{}
	})
}

func ioShapingPolicyTickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return ioShapingPolicyTickMsg{}
	})
}

func loadLogCmd(client *eos.Client, host, filePath string) tea.Cmd {
	return func() tea.Msg {
		out, err := client.TailLogOnHost(context.Background(), host, filePath, 2000)
		if err != nil {
			return logLoadedMsg{filePath: filePath, err: err}
		}
		raw := strings.TrimRight(string(out), "\n")
		lines := strings.Split(raw, "\n")
		return logLoadedMsg{filePath: filePath, lines: lines}
	}
}

func logTickCmd() tea.Cmd {
	return tea.Tick(logRefreshInterval, func(time.Time) tea.Msg {
		return logTickMsg{}
	})
}

func loadCommandHistoryCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return commandHistoryLoadedMsg{err: fmt.Errorf("command logging unavailable")}
		}
		lines, err := client.SessionCommands(commandLogTailLines)
		return commandHistoryLoadedMsg{
			filePath: client.SessionLogPath(),
			lines:    lines,
			err:      err,
		}
	}
}

func commandLogTickCmd() tea.Cmd {
	return tea.Tick(commandLogRefreshInterval, func(time.Time) tea.Msg {
		return commandLogTickMsg{}
	})
}

func splashTickCmd() tea.Cmd {
	return tea.Tick(startupSplashTickInterval, func(time.Time) tea.Msg {
		return splashTickMsg{}
	})
}
