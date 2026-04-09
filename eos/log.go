package eos

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func (c *Client) openLogFile() (*os.File, error) {
	if c.sessionLogPath == "" {
		return nil, fmt.Errorf("logging disabled")
	}
	return os.OpenFile(c.sessionLogPath, os.O_APPEND|os.O_WRONLY, 0644)
}

func (c *Client) logCommand(args []string) {
	f, err := c.openLogFile()
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	var command string
	if target := c.effectiveSSHTarget(); target != "" {
		// Log as a fully copy-pasteable SSH invocation.
		remoteCmd := strings.Join(args, " ")
		command = fmt.Sprintf("ssh -o BatchMode=yes %s %s", target, shellQuote(remoteCmd))
	} else {
		command = strings.Join(args, " ")
	}
	_, _ = f.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, command))
}

func (c *Client) logResponse(args []string, output []byte, err error) {
	f, ferr := c.openLogFile()
	if ferr != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	// Abbreviate very long output to avoid flooding the log.
	preview := strings.TrimSpace(string(output))
	const maxPreview = 500
	if len(preview) > maxPreview {
		preview = preview[:maxPreview] + "...(truncated)"
	}
	var cmdStr string
	if len(args) > 0 {
		cmdStr = args[len(args)-1] // last arg as a short label
	}
	_, _ = f.WriteString(fmt.Sprintf("[%s] ERROR (%s): %v\n", timestamp, cmdStr, err))
	if preview != "" {
		_, _ = f.WriteString(fmt.Sprintf("[%s]   output: %s\n", timestamp, preview))
	}
}
