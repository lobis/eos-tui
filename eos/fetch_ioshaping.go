package eos

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Client) IOShaping(ctx context.Context, mode IOShapingMode) ([]IOShapingRecord, error) {
	flag := "--apps"
	switch mode {
	case IOShapingUsers:
		flag = "--users"
	case IOShapingGroups:
		flag = "--groups"
	}
	output, err := c.runCommand("eos", "io", "shaping", "ls", flag, "--json", "--window", "5")
	if err != nil {
		return nil, fmt.Errorf("io shaping ls: %w: %s", err, strings.TrimSpace(string(output)))
	}

	var raw []struct {
		ID        string  `json:"id"`
		Type      string  `json:"type"`
		WindowSec int     `json:"window_sec"`
		ReadBps   float64 `json:"read_rate_bps"`
		WriteBps  float64 `json:"write_rate_bps"`
		ReadIOPS  float64 `json:"read_iops"`
		WriteIOPS float64 `json:"write_iops"`
	}
	if err := json.Unmarshal(stripEOSPreamble(output), &raw); err != nil {
		return nil, fmt.Errorf("parse io shaping: %w", err)
	}

	records := make([]IOShapingRecord, len(raw))
	for i, r := range raw {
		records[i] = IOShapingRecord{
			ID:        r.ID,
			Type:      r.Type,
			WindowSec: r.WindowSec,
			ReadBps:   r.ReadBps,
			WriteBps:  r.WriteBps,
			ReadIOPS:  r.ReadIOPS,
			WriteIOPS: r.WriteIOPS,
		}
	}
	return records, nil
}

func (c *Client) IOShapingPolicies(ctx context.Context) ([]IOShapingPolicyRecord, error) {
	_ = ctx
	output, err := c.runCommand("eos", "io", "shaping", "policy", "ls", "--json")
	if err != nil {
		return nil, fmt.Errorf("io shaping policy ls: %w: %s", err, strings.TrimSpace(string(output)))
	}

	var raw []struct {
		ID                          string  `json:"id"`
		Type                        string  `json:"type"`
		Enabled                     bool    `json:"is_enabled"`
		LimitReadBytesPerSec        float64 `json:"limit_read_bytes_per_sec"`
		LimitWriteBytesPerSec       float64 `json:"limit_write_bytes_per_sec"`
		ReservationReadBytesPerSec  float64 `json:"reservation_read_bytes_per_sec"`
		ReservationWriteBytesPerSec float64 `json:"reservation_write_bytes_per_sec"`
	}
	if err := json.Unmarshal(stripEOSPreamble(output), &raw); err != nil {
		return nil, fmt.Errorf("parse io shaping policy: %w", err)
	}

	records := make([]IOShapingPolicyRecord, len(raw))
	for i, r := range raw {
		records[i] = IOShapingPolicyRecord{
			ID:                          r.ID,
			Type:                        r.Type,
			Enabled:                     r.Enabled,
			LimitReadBytesPerSec:        r.LimitReadBytesPerSec,
			LimitWriteBytesPerSec:       r.LimitWriteBytesPerSec,
			ReservationReadBytesPerSec:  r.ReservationReadBytesPerSec,
			ReservationWriteBytesPerSec: r.ReservationWriteBytesPerSec,
		}
	}
	return records, nil
}
