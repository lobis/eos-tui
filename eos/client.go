package eos

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func New(_ context.Context, cfg Config) (*Client, error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	c := &Client{
		sshTarget:         cfg.SSHTarget,
		timeout:           timeout,
		acceptNewHostKeys: cfg.AcceptNewHostKeys,
	}
	c.sessionLogPath = initSessionLog()
	c.supportsNSSnapshotTopology()
	return c, nil
}

// initSessionLog creates ~/.eos-tui/sessions/ if needed, generates a
// timestamped log file path for this session, and updates the
// ~/.eos-tui/latest.log symlink to point at it.
// Returns the session log path, or "" if setup fails (logging silently disabled).
func initSessionLog() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	logDir := filepath.Join(home, ".eos-tui", "sessions")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return ""
	}

	// Use a timestamp that is both human-readable and filesystem-safe.
	ts := time.Now().Format("2006-01-02T15-04-05")
	sessionFile := filepath.Join(logDir, ts+".log")

	// Create the file immediately so the symlink target exists.
	f, err := os.OpenFile(sessionFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return ""
	}
	f.Close()

	// Update ~/.eos-tui/latest.log → sessions/<timestamp>.log (relative symlink).
	latestLink := filepath.Join(home, ".eos-tui", "latest.log")
	// Relative target from ~/.eos-tui/ to sessions/<ts>.log
	relTarget := filepath.Join("sessions", ts+".log")
	// Remove stale symlink (or file) then re-create.
	_ = os.Remove(latestLink)
	_ = os.Symlink(relTarget, latestLink)

	return sessionFile
}

func (c *Client) Close() error {
	return nil
}

// effectiveSSHTarget returns the host that runCommand will actually SSH to.
func (c *Client) effectiveSSHTarget() string {
	if c.resolvedSSHTarget != "" {
		return c.resolvedSSHTarget
	}
	return c.sshTarget
}

// ResolvedSSHTarget returns the effective SSH target after master discovery,
// or the original target if discovery has not run.
func (c *Client) ResolvedSSHTarget() string {
	return c.effectiveSSHTarget()
}

// OriginalSSHTarget returns the user-supplied SSH target before master discovery.
func (c *Client) OriginalSSHTarget() string {
	return c.sshTarget
}

// ensureRootPrefix returns target with a "root@" prefix, adding one only if
// it is not already present.
func ensureRootPrefix(target string) string {
	if strings.HasPrefix(target, "root@") {
		return target
	}
	return "root@" + target
}

func (c *Client) supportsNSSnapshotTopology() bool {
	c.nsSnapshotSupport.once.Do(func() {
		output, err := c.runCommand("eos", "ns", "snapshot")
		if err != nil {
			c.nsSnapshotSupport.supported = false
			return
		}
		c.nsSnapshotSupport.supported = hasNSSnapshotTopology(output)
	})
	return c.nsSnapshotSupport.supported
}

// DiscoverMGMMaster prefers the cached `eos ns snapshot` topology on the
// current SSH target and falls back to `redis-cli raft-info` when the newer
// namespace output is not available yet. It identifies the MGM leader and
// updates the client so that all subsequent commands are routed directly to it.
// Returns the resolved hostname (e.g. "eospilot-ns-02.cern.ch").
func (c *Client) DiscoverMGMMaster(ctx context.Context) (string, error) {
	_ = ctx

	if c.supportsNSSnapshotTopology() {
		output, err := c.runCommand("eos", "ns", "snapshot")
		if err == nil {
			values := parseMonitoringAssignments(output)
			if leader := values["ns.mgm.leader"]; leader != "" {
				resolved := ensureRootPrefix(hostOnly(leader))
				c.resolvedSSHTarget = resolved
				return resolved, nil
			}
		}
	}

	output, err := c.runCommand("redis-cli", "-p", "7777", "raft-info")
	if err != nil {
		return "", fmt.Errorf("raft-info for master discovery: %w", err)
	}

	// QDB may require authentication — fall back to the current SSH target.
	if strings.Contains(string(output), "NOAUTH") {
		target := c.effectiveSSHTarget()
		if target == "" {
			return "", fmt.Errorf("raft-info requires authentication and no SSH target is configured")
		}
		resolved := ensureRootPrefix(target)
		c.resolvedSSHTarget = resolved
		return resolved, nil
	}

	info := parseRaftInfo(output)
	if info.Leader == "" {
		return "", fmt.Errorf("no leader found in raft-info output")
	}

	leader := hostOnly(info.Leader)
	// EOS nodes run as root; use explicit root@ so the resolved hostname
	// works without relying on SSH config aliases.
	resolved := ensureRootPrefix(leader)
	c.resolvedSSHTarget = resolved
	return resolved, nil
}
