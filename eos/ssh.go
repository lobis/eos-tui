package eos

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func (c *Client) sshOptions(batchMode bool) []string {
	options := []string{"-o", "LogLevel=ERROR"}
	if batchMode {
		options = append(options, "-o", "BatchMode=yes")
	} else {
		options = append(options, "-o", "BatchMode=no")
	}
	if c.acceptNewHostKeys {
		options = append(options, "-o", "StrictHostKeyChecking=accept-new")
	}
	return options
}

func (c *Client) SSHArgs(batchMode bool, extraArgs ...string) []string {
	args := c.sshOptions(batchMode)
	args = append(args, extraArgs...)
	return args
}

func (c *Client) runCommand(args ...string) ([]byte, error) {
	c.logCommand(args)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	target := c.effectiveSSHTarget()
	var out []byte
	var err error
	if target == "" {
		out, err = exec.CommandContext(ctx, args[0], args[1:]...).CombinedOutput()
	} else {
		remoteCommand := shellJoin(args)
		sshArgs := append(c.SSHArgs(true), target, remoteCommand)
		out, err = exec.CommandContext(ctx, "ssh", sshArgs...).CombinedOutput()
	}

	if err != nil {
		c.logResponse(args, out, err)
	}
	return out, err
}

func (c *Client) runCommandOnHost(ctx context.Context, host string, args ...string) ([]byte, error) {
	host = strings.TrimSpace(host)
	if host == "" || host == hostOnly(strings.TrimPrefix(c.effectiveSSHTarget(), "root@")) {
		return c.runCommand(args...)
	}

	target := ensureRootPrefix(host)
	remoteCommand := shellJoin(args)
	c.logCommand(append([]string{"→", target}, args...))

	ctxTimeout, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	effective := c.effectiveSSHTarget()
	var sshArgs []string
	if effective == "" {
		sshArgs = append(c.SSHArgs(true), target, remoteCommand)
	} else {
		sshArgs = append(c.SSHArgs(true), "-J", effective, target, remoteCommand)
	}

	out, err := exec.CommandContext(ctxTimeout, "ssh", sshArgs...).CombinedOutput()
	if err != nil {
		c.logResponse(args, out, err)
	}
	return out, err
}

// RTLog queries EOS real-time logs through the CLI instead of tailing a
// specific logfile path. The queue identifies either the MGM (".") or an FST
// queue such as /eos/host:port/fst.
func (c *Client) RTLog(ctx context.Context, queue string, seconds int, tag string) ([]byte, error) {
	if queue == "" {
		queue = "."
	}
	if seconds <= 0 {
		seconds = 600
	}
	if tag == "" {
		tag = "info"
	}

	args := []string{"eos", "rtlog", queue, fmt.Sprintf("%d", seconds), tag}
	out, err := c.runCommand(args...)
	if err != nil {
		return nil, fmt.Errorf("eos rtlog %s %d %s: %w (output: %.300s)", queue, seconds, tag, err, out)
	}
	return out, nil
}

// TailLog returns the last n lines of a log file on the effective SSH target
// (or locally when no SSH target is configured).
func (c *Client) TailLog(ctx context.Context, filePath string, n int) ([]byte, error) {
	return c.TailLogOnHost(ctx, "", filePath, n)
}

// TailLogOnHost returns the last n lines of a log file on a specific host.
// When host is empty or matches the current effective target it is equivalent
// to TailLog.  Otherwise the command is routed to the named host, using the
// effective SSH target as a jump proxy when one is configured.
func (c *Client) TailLogOnHost(ctx context.Context, host, filePath string, n int) ([]byte, error) {
	tailArgs := []string{"tail", fmt.Sprintf("-n%d", n), filePath}

	effective := c.effectiveSSHTarget() // e.g. "root@eospilot-ns-02.cern.ch"
	effectiveHost := hostOnly(strings.TrimPrefix(effective, "root@"))

	// Direct case: no specific host, or the host IS the current target.
	if host == "" || host == effectiveHost {
		out, err := c.runCommand(tailArgs...)
		if err != nil {
			return nil, fmt.Errorf("tail %s: %w (output: %.300s)", filePath, err, out)
		}
		return out, nil
	}

	// We need to reach a different host.  Use the effective target as a jump
	// proxy (or SSH directly when running locally).
	target := "root@" + host
	tailCmd := shellJoin(tailArgs)

	c.logCommand(append([]string{"→", target}, tailArgs...))
	ctxTimeout, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var out []byte
	var err error
	if effective == "" {
		sshArgs := append(c.SSHArgs(true), target, tailCmd)
		out, err = exec.CommandContext(ctxTimeout, "ssh", sshArgs...).CombinedOutput()
	} else {
		sshArgs := append(c.SSHArgs(true), "-J", effective, target, tailCmd)
		out, err = exec.CommandContext(ctxTimeout, "ssh", sshArgs...).CombinedOutput()
	}
	if err != nil {
		c.logResponse(tailArgs, out, err)
		return nil, fmt.Errorf("tail %s on %s: %w (output: %.300s)", filePath, host, err, out)
	}
	return out, nil
}

// SSHTargetForHost returns the ssh arguments needed to open an interactive
// shell on host, routing via the effective SSH target when necessary.
// Returns (directTarget, jumpProxy) where jumpProxy may be empty.
func (c *Client) SSHTargetForHost(host string) (target, jump string) {
	effective := c.effectiveSSHTarget()
	effectiveHost := hostOnly(strings.TrimPrefix(effective, "root@"))

	if host == "" || host == effectiveHost {
		if effective != "" {
			return effective, ""
		}
		return "", ""
	}

	target = "root@" + host
	if effective != "" {
		return target, effective
	}
	return target, ""
}

func (c *Client) AcceptNewHostKeys() bool {
	return c.acceptNewHostKeys
}
