package eos

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) EOSVersion(ctx context.Context) (string, error) {
	_ = ctx
	output, err := c.runCommand("eos", "version")
	if err != nil {
		return "", fmt.Errorf("eos version: %w", err)
	}
	return parseEOSServerVersion(output), nil
}

// parseEOSServerVersion extracts the server version from `eos version` output.
// Example line: "EOS_SERVER_VERSION=5.3.27 EOS_SERVER_RELEASE=unknown"
func parseEOSServerVersion(output []byte) string {
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "EOS_SERVER_VERSION=") {
			rest := strings.TrimPrefix(line, "EOS_SERVER_VERSION=")
			if fields := strings.Fields(rest); len(fields) > 0 {
				return fields[0]
			}
		}
	}
	return ""
}
