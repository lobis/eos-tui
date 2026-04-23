package eos

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) AccessList(ctx context.Context) ([]AccessRecord, error) {
	output, err := c.runCommandContext(ctx, "eos", "access", "ls", "-m")
	if err != nil {
		return nil, fmt.Errorf("eos access ls -m: %w", err)
	}

	return parseAccessList(output), nil
}

func (c *Client) SetAccessRule(ctx context.Context, op, category, value string) error {
	args, err := accessRuleArgs(op, category, value)
	if err != nil {
		return err
	}
	if _, err := c.runCommandContext(ctx, args...); err != nil {
		return fmt.Errorf("%s %s %s: %w", strings.Join(args[:3], " "), category, value, err)
	}
	return nil
}

func (c *Client) SetAccessStall(ctx context.Context, seconds int) error {
	args, err := accessStallArgs(seconds)
	if err != nil {
		return err
	}
	if _, err := c.runCommandContext(ctx, args...); err != nil {
		return fmt.Errorf("%s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func parseAccessList(output []byte) []AccessRecord {
	lines := strings.Split(string(output), "\n")
	records := make([]AccessRecord, 0, len(lines))
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		rawKey, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		rawKey = strings.TrimSpace(rawKey)
		value = strings.TrimSpace(value)
		category, rule, hasRule := strings.Cut(rawKey, ".")
		if !hasRule {
			category = rawKey
			rule = "value"
		}

		records = append(records, AccessRecord{
			Category: category,
			Rule:     rule,
			Value:    value,
			RawKey:   rawKey,
		})
	}

	return records
}

func accessRuleArgs(op, category, value string) ([]string, error) {
	op = strings.TrimSpace(strings.ToLower(op))
	category = strings.TrimSpace(strings.ToLower(category))
	value = strings.TrimSpace(value)

	switch op {
	case "allow", "unallow", "ban", "unban":
	default:
		return nil, fmt.Errorf("unsupported access action %q", op)
	}

	switch category {
	case "user", "group", "host", "domain":
	default:
		return nil, fmt.Errorf("unsupported access category %q", category)
	}

	if value == "" {
		return nil, fmt.Errorf("access value is required")
	}

	return []string{"eos", "access", op, category, value}, nil
}

func accessStallArgs(seconds int) ([]string, error) {
	if seconds <= 0 {
		return nil, fmt.Errorf("stall seconds must be positive")
	}
	return []string{"eos", "access", "set", "stall", fmt.Sprintf("%d", seconds)}, nil
}
