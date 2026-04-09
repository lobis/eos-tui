package eos

import (
	"bufio"
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

func (c *Client) SessionLogPath() string {
	return c.sessionLogPath
}

func (c *Client) SessionCommands(n int) ([]string, error) {
	if c.sessionLogPath == "" {
		return nil, fmt.Errorf("logging disabled")
	}
	if n <= 0 {
		return nil, nil
	}

	f, err := os.Open(c.sessionLogPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lines := make([]string, 0, n)
	for scanner.Scan() {
		line := scanner.Text()
		if !isSessionCommandLine(line) {
			continue
		}
		if len(lines) == n {
			copy(lines, lines[1:])
			lines = lines[:n-1]
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func isSessionCommandLine(line string) bool {
	if !strings.HasPrefix(line, "[") {
		return false
	}
	if strings.Contains(line, "] ERROR ") {
		return false
	}
	if strings.Contains(line, "]   output: ") {
		return false
	}
	return true
}

// LogCommand writes an arbitrary command line to the session log in the same
// format used by runCommand. Use this for commands issued outside of the
// normal eos.Client SSH path (e.g. direct exec.Command calls from the UI).
func (c *Client) LogCommand(args []string) {
	c.logCommand(args)
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
		command = fmt.Sprintf("ssh -o BatchMode=yes %s %s", target, shellDisplayJoin(args))
	} else {
		command = shellDisplayJoin(args)
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
