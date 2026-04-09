package eos

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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

func (c *Client) SetIOShapingPolicy(ctx context.Context, update IOShapingPolicyUpdate) error {
	_ = ctx

	args, err := ioShapingPolicySetArgs(update)
	if err != nil {
		return err
	}

	if _, err := c.runCommand(args...); err != nil {
		return fmt.Errorf("eos io shaping policy set %s: %w", update.ID, err)
	}
	return nil
}

func (c *Client) RemoveIOShapingPolicy(ctx context.Context, mode IOShapingMode, id string) error {
	_ = ctx

	args, err := ioShapingPolicyRemoveArgs(mode, id)
	if err != nil {
		return err
	}
	if _, err := c.runCommand(args...); err != nil {
		return fmt.Errorf("eos io shaping policy rm %s: %w", id, err)
	}
	return nil
}

func ioShapingPolicySetArgs(update IOShapingPolicyUpdate) ([]string, error) {
	targetFlag, err := ioShapingPolicyTargetFlag(update.Mode)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(update.ID) == "" {
		return nil, fmt.Errorf("io shaping policy id is required")
	}

	args := []string{
		"eos", "io", "shaping", "policy", "set",
		targetFlag, update.ID,
	}
	if update.Enabled {
		args = append(args, "--enable")
	} else {
		args = append(args, "--disable")
	}
	args = append(args,
		"--limit-read", strconv.FormatUint(update.LimitReadBytesPerSec, 10),
		"--limit-write", strconv.FormatUint(update.LimitWriteBytesPerSec, 10),
		"--reservation-read", strconv.FormatUint(update.ReservationReadBytesPerSec, 10),
		"--reservation-write", strconv.FormatUint(update.ReservationWriteBytesPerSec, 10),
	)
	return args, nil
}

func ioShapingPolicyTargetFlag(mode IOShapingMode) (string, error) {
	switch mode {
	case IOShapingApps:
		return "--app", nil
	case IOShapingUsers:
		return "--uid", nil
	case IOShapingGroups:
		return "--gid", nil
	default:
		return "", fmt.Errorf("unsupported io shaping mode %d", mode)
	}
}

func ioShapingPolicyRemoveArgs(mode IOShapingMode, id string) ([]string, error) {
	targetFlag, err := ioShapingPolicyTargetFlag(mode)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("io shaping policy id is required")
	}

	return []string{
		"eos", "io", "shaping", "policy", "rm",
		targetFlag, id,
	}, nil
}
