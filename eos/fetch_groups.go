package eos

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
)

func (c *Client) Groups(ctx context.Context) ([]GroupRecord, error) {
	_ = ctx

	output, err := c.runCommand("eos", "-j", "-b", "group", "ls")
	if err != nil {
		return nil, fmt.Errorf("eos group ls: %w", err)
	}

	var payload struct {
		Result []struct {
			Name string `json:"name"`
			NoFS int    `json:"nofs"`
			Cfg  struct {
				Status string `json:"status"`
			} `json:"cfg"`
			Sum struct {
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
		return nil, fmt.Errorf("parse group ls: %w (output: %.200s)", err, output)
	}

	groups := make([]GroupRecord, 0, len(payload.Result))
	for _, item := range payload.Result {
		groups = append(groups, GroupRecord{
			Name:          item.Name,
			Status:        item.Cfg.Status,
			NoFS:          item.NoFS,
			CapacityBytes: item.Sum.Stat.StatFS.Capacity,
			UsedBytes:     item.Sum.Stat.StatFS.UsedBytes,
			FreeBytes:     item.Sum.Stat.StatFS.FreeBytes,
			NumFiles:      item.Sum.Stat.StatFS.Files,
		})
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Name < groups[j].Name
	})

	return groups, nil
}
