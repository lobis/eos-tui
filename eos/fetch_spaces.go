package eos

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func (c *Client) Spaces(ctx context.Context) ([]SpaceRecord, error) {
	_ = ctx

	output, err := c.runCommand("eos", "-j", "-b", "space", "ls")
	if err != nil {
		return nil, fmt.Errorf("eos space ls: %w", err)
	}

	var payload struct {
		Result []struct {
			Name string `json:"name"`
			Type string `json:"type"`
			Cfg  struct {
				GroupSize uint64 `json:"groupsize"`
			} `json:"cfg"`
			Sum struct {
				NRW  uint64 `json:"n_rw"`
				Stat struct {
					StatFS struct {
						Capacity  uint64 `json:"capacity"`
						UsedBytes uint64 `json:"usedbytes"`
						FreeBytes uint64 `json:"freebytes"`
						Files     uint64 `json:"files"`
					} `json:"statfs"`
				} `json:"stat"`
			} `json:"sum"`
		} `json:"result"`
	}

	if err := json.Unmarshal(stripEOSPreamble(output), &payload); err != nil {
		return nil, fmt.Errorf("parse space ls: %w (output: %.200s)", err, output)
	}

	spaces := make([]SpaceRecord, 0, len(payload.Result))
	for _, item := range payload.Result {
		spaces = append(spaces, SpaceRecord{
			Name:          item.Name,
			Type:          item.Type,
			Status:        "active",
			Groups:        item.Cfg.GroupSize,
			NumFiles:      item.Sum.Stat.StatFS.Files,
			NumContainers: item.Sum.NRW,
			CapacityBytes: item.Sum.Stat.StatFS.Capacity,
			UsedBytes:     item.Sum.Stat.StatFS.UsedBytes,
			FreeBytes:     item.Sum.Stat.StatFS.FreeBytes,
		})
	}

	sort.Slice(spaces, func(i, j int) bool {
		return spaces[i].Name < spaces[j].Name
	})

	return spaces, nil
}

func (c *Client) SpaceStatus(ctx context.Context, name string) ([]SpaceStatusRecord, error) {
	_ = ctx

	// TODO: use JSON output once it is reliable.
	// output, err := c.runCommand("eos", "-j", "-b", "space", "status", name)
	output, err := c.runCommand("eos", "-b", "space", "status", name)
	if err != nil {
		return nil, fmt.Errorf("eos space status %s: %w", name, err)
	}

	return parseSpaceStatus(output), nil
}

func (c *Client) SpaceConfig(ctx context.Context, name string, key, value string) error {
	_ = ctx

	fullKey := key
	if !strings.HasPrefix(key, "space.") && !strings.HasPrefix(key, "fs.") {
		fullKey = "space." + key
	}

	_, err := c.runCommand("eos", "-b", "space", "config", name, fmt.Sprintf("%s=%s", fullKey, value))
	if err != nil {
		return fmt.Errorf("eos space config %s %s=%s: %w", name, fullKey, value, err)
	}

	return nil
}

func parseSpaceStatus(output []byte) []SpaceStatusRecord {
	lines := strings.Split(string(output), "\n")
	records := make([]SpaceStatusRecord, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":=", 2)
		if len(parts) != 2 {
			continue
		}

		records = append(records, SpaceStatusRecord{
			Key:   strings.TrimSpace(parts[0]),
			Value: strings.TrimSpace(parts[1]),
		})
	}

	return records
}
