package eos

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// VIDEntries returns the list of virtual identity mappings from `eos vid ls`.
// It first attempts JSON output; if that fails it falls back to text parsing.
func (c *Client) VIDEntries(ctx context.Context) ([]VIDRecord, error) {
	_ = ctx

	output, err := c.runCommand("eos", "-j", "-b", "vid", "ls")
	if err == nil {
		records, parseErr := parseVIDJSON(output)
		if parseErr == nil {
			return records, nil
		}
	}

	// Fallback: text format.
	output, err = c.runCommand("eos", "-b", "vid", "ls")
	if err != nil {
		return nil, fmt.Errorf("eos vid ls: %w", err)
	}

	return parseVIDText(output), nil
}

// parseVIDJSON parses the JSON output of `eos -j vid ls`.
func parseVIDJSON(output []byte) ([]VIDRecord, error) {
	var payload struct {
		Result []struct {
			Auth  string `json:"auth"`
			Match string `json:"match"`
			UID   string `json:"uid"`
			GID   string `json:"gid"`
		} `json:"result"`
	}

	if err := json.Unmarshal(stripEOSPreamble(output), &payload); err != nil {
		return nil, fmt.Errorf("parse vid ls json: %w (output: %.200s)", err, output)
	}

	records := make([]VIDRecord, 0, len(payload.Result))
	for _, item := range payload.Result {
		if item.Auth == "" && item.Match == "" {
			continue
		}
		records = append(records, VIDRecord{
			Auth:  item.Auth,
			Match: item.Match,
			UID:   item.UID,
			GID:   item.GID,
		})
	}

	return records, nil
}

// parseVIDText parses the human-readable output of `eos vid ls`.
// Lines look like:
//
//	tident: user@[127.0.0.1]:<0>  =>  uid=0 gid=0
//	host:   localhost              =>  uid=0 gid=0
func parseVIDText(output []byte) []VIDRecord {
	lines := strings.Split(string(output), "\n")
	records := make([]VIDRecord, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on the "=>" arrow separating physical from virtual identity.
		arrowParts := strings.SplitN(line, "=>", 2)
		if len(arrowParts) != 2 {
			continue
		}

		lhs := strings.TrimSpace(arrowParts[0])
		rhs := strings.TrimSpace(arrowParts[1])

		// lhs format: "auth: match"
		colonIdx := strings.Index(lhs, ":")
		if colonIdx < 0 {
			continue
		}
		auth := strings.TrimSpace(lhs[:colonIdx])
		match := strings.TrimSpace(lhs[colonIdx+1:])

		if auth == "" {
			continue
		}

		// rhs format: "uid=X gid=Y" or "uid:X gid:Y"
		rhs = strings.ReplaceAll(rhs, "uid:", "uid=")
		rhs = strings.ReplaceAll(rhs, "gid:", "gid=")
		uid := ""
		gid := ""
		for _, field := range strings.Fields(rhs) {
			switch {
			case strings.HasPrefix(field, "uid="):
				uid = strings.TrimPrefix(field, "uid=")
			case strings.HasPrefix(field, "gid="):
				gid = strings.TrimPrefix(field, "gid=")
			}
		}

		records = append(records, VIDRecord{
			Auth:  auth,
			Match: match,
			UID:   uid,
			GID:   gid,
		})
	}

	return records
}
