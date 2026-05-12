package eos

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) EOSVersion(ctx context.Context) (string, error) {
	output, err := c.runCommandContext(ctx, "eos", "version")
	if err != nil {
		return "", fmt.Errorf("eos version: %w", err)
	}
	return parseEOSServerVersion(output), nil
}

func parseEOSServerBuildVersion(output []byte) string {
	values := parseMonitoringKeyValues(output)
	version := strings.TrimSpace(values["EOS_SERVER_VERSION"])
	release := strings.TrimSpace(values["EOS_SERVER_RELEASE"])
	if version != "" && release != "" && release != "unknown" {
		return version + "-" + release
	}
	if version != "" {
		return version
	}
	return parseEOSServerVersion(output)
}

func parseEOSServerPackageVersion(output []byte) string {
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "eos-server-")
		line = strings.TrimSuffix(line, ".x86_64")
		if line == "" || strings.HasPrefix(line, "package ") {
			continue
		}
		if before, _, ok := strings.Cut(line, ".el"); ok {
			line = before
		}
		return line
	}
	return ""
}

// parseEOSServerVersion extracts the server version from either `eos version`
// output (EOS_SERVER_VERSION=...) or `eos --version` output (EOS x.y.z (...)).
func parseEOSServerVersion(output []byte) string {
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "EOS_SERVER_VERSION=") {
			rest := strings.TrimPrefix(line, "EOS_SERVER_VERSION=")
			if fields := strings.Fields(rest); len(fields) > 0 {
				return fields[0]
			}
		}
		if strings.HasPrefix(line, "EOS ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return fields[1]
			}
		}
	}
	return ""
}
