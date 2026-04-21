package eos

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) VIDList(ctx context.Context, flag string) ([]VIDRecord, error) {
	_ = ctx

	args := []string{"eos", "vid", "ls"}
	flag = strings.TrimSpace(flag)
	if flag != "" {
		args = append(args, flag)
	}

	output, err := c.runCommand(args...)
	if err != nil {
		command := "eos vid ls"
		if flag != "" {
			command += " " + flag
		}
		return nil, fmt.Errorf("%s: %w", command, err)
	}

	return parseVIDList(output), nil
}

func parseVIDList(output []byte) []VIDRecord {
	lines := strings.Split(string(output), "\n")
	records := make([]VIDRecord, 0, len(lines))
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "* ") {
			continue
		}

		key, value, found := strings.Cut(line, "=>")
		if !found {
			records = append(records, VIDRecord{Key: strings.TrimSuffix(line, ":")})
			continue
		}

		records = append(records, VIDRecord{
			Key:   strings.TrimSuffix(strings.TrimSpace(key), ":"),
			Value: strings.TrimSpace(value),
		})
	}

	return records
}
