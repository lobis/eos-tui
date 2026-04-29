package eos

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) QDBAttemptCoup(ctx context.Context, host string) ([]byte, error) {
	args := []string{"redis-cli", "-p", "7777", "raft-attempt-coup"}
	out, err := c.runCommandOnHost(ctx, host, args...)
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail != "" {
			return out, fmt.Errorf("redis-cli raft-attempt-coup on %s: %w\n%s", host, err, detail)
		}
		return out, fmt.Errorf("redis-cli raft-attempt-coup on %s: %w", host, err)
	}
	return out, nil
}
