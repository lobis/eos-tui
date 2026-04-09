package eos

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func (c *Client) MGMs(ctx context.Context) ([]MgmRecord, error) {
	_ = ctx

	// Run redis-cli raft-info directly via runCommand.
	// The SSH target (if set) is always the MGM or an MGM leader node,
	// so we do not need a separate SSH hop.
	output, err := c.runCommand("redis-cli", "-p", "7777", "raft-info")
	if err != nil {
		return nil, fmt.Errorf("redis-cli raft-info: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	// QDB may be configured to require authentication (returns "NOAUTH ...").
	// In that case fall back to treating the current SSH target as the single
	// MGM leader — enough to satisfy callers when the cluster is reachable but
	// the raw redis port is not open to unauthenticated clients.
	if strings.Contains(string(output), "NOAUTH") {
		return c.mgmsFromSSHTarget()
	}

	info := parseRaftInfo(output)

	if info.Leader == "" && len(info.Nodes) == 0 && info.Myself == "" {
		return nil, fmt.Errorf("no MGM cluster info from raft-info")
	}

	// Fetch the MGM service port from `eos ns stat` via master_id
	// (e.g. "eospilot-ns-02.cern.ch:1094"). The raft nodes use the QDB port
	// (7777); the actual MGM port must be read from the namespace.
	mgmPort := mgmPortFromNsStat(c)

	// Determine leader hostname (strip raft port :7777)
	leaderHost := hostOnly(info.Leader)

	// Build replica status/version map keyed by hostname.
	replicaMap := make(map[string]raftReplica)
	for _, r := range info.Replicas {
		replicaMap[hostOnly(r.Host)] = r
	}

	// Use NODES list; fall back to just MYSELF if NODES is empty.
	nodes := info.Nodes
	if len(nodes) == 0 && info.Myself != "" {
		nodes = []string{info.Myself}
	}

	seen := make(map[string]bool)
	mgms := make([]MgmRecord, 0, len(nodes))

	for _, node := range nodes {
		host := hostOnly(node)
		if seen[host] {
			continue
		}
		seen[host] = true

		role := "follower"
		if host == leaderHost {
			role = "leader"
		}

		status := "online"
		version := ""

		if r, ok := replicaMap[host]; ok {
			if strings.ToUpper(r.Status) == "OFFLINE" {
				status = "offline"
			}
			version = r.Version
		}

		// The leader runs as MYSELF — use the version from that section.
		if host == leaderHost && info.MyVersion != "" {
			version = info.MyVersion
		}

		h, p := splitHostPort(host + ":" + mgmPort)
		qh, qp := splitHostPort(node)

		mgms = append(mgms, MgmRecord{
			Host:       h,
			Port:       p,
			QDBHost:    qh,
			QDBPort:    qp,
			Role:       role,
			Status:     status,
			EOSVersion: version,
		})
	}

	// Sort: leader first, then alphabetically.
	sort.Slice(mgms, func(i, j int) bool {
		if mgms[i].Role != mgms[j].Role {
			return mgms[i].Role == "leader"
		}
		if mgms[i].Host != mgms[j].Host {
			return mgms[i].Host < mgms[j].Host
		}
		return mgms[i].Port < mgms[j].Port
	})

	return mgms, nil
}

// mgmPortFromNsStat fetches the MGM service port by reading master_id from
// `eos ns stat`.  master_id is of the form "hostname:port" (e.g.
// "eospilot-ns-02.cern.ch:1094"). Falls back to "1094" on any error.
func mgmPortFromNsStat(c *Client) string {
	const fallback = "1094"

	out, err := c.runCommand("eos", "-j", "-b", "ns", "stat")
	if err != nil {
		return fallback
	}

	var payload struct {
		Result []struct {
			Master string `json:"master_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(stripEOSPreamble(out), &payload); err != nil || len(payload.Result) == 0 {
		return fallback
	}

	masterID := payload.Result[0].Master // e.g. "eospilot-ns-02.cern.ch:1094"
	if idx := strings.LastIndex(masterID, ":"); idx != -1 {
		if port := masterID[idx+1:]; port != "" {
			return port
		}
	}
	return fallback
}

// mgmsFromSSHTarget constructs a minimal single-entry MGM list from the
// current effective SSH target.  It is used as a fallback when redis-cli
// raft-info is unavailable (e.g. QDB requires authentication).  The caller
// is assumed to already be connected to an MGM node, so that node is treated
// as the cluster leader.
func (c *Client) mgmsFromSSHTarget() ([]MgmRecord, error) {
	target := c.effectiveSSHTarget()
	if target == "" {
		return nil, fmt.Errorf("redis-cli raft-info requires authentication and no SSH target is configured")
	}
	// Strip optional "root@" prefix so we work with a plain host[:port] string.
	host := strings.TrimPrefix(target, "root@")
	h, p := splitHostPort(host)
	if p == 0 {
		p = 1094
	}
	return []MgmRecord{{
		Host:   h,
		Port:   p,
		Role:   "leader",
		Status: "online",
	}}, nil
}

// parseRaftInfo parses the output of `redis-cli -p 7777 raft-info`.
//
// Example output (multi-node):
//
//	LEADER   eospilot-ns-02.cern.ch:7777
//	MYSELF   eospilot-ns-02.cern.ch:7777
//	STATUS   LEADER
//	VERSION  5.3.29.1
//	NODE-HEALTH GREEN
//	NODES    eospilot-ns-ip700.cern.ch:7777,eospilot-ns-01.cern.ch:7777,...
//	REPLICA  eospilot-ns-01.cern.ch:7777 | ONLINE | UP-TO-DATE | LOG-SIZE ... | VERSION 5.3.29.1
func parseRaftInfo(output []byte) raftInfo {
	var info raftInfo
	inMyselfSection := false

	for _, raw := range strings.Split(string(output), "\n") {
		line := strings.TrimSpace(raw)
		if line == "----------" {
			inMyselfSection = false
			continue
		}

		// Each line is "KEY value" separated by whitespace.
		idx := strings.IndexAny(line, " \t")
		if idx < 0 {
			continue
		}
		key := line[:idx]
		value := strings.TrimSpace(line[idx+1:])

		switch key {
		case "LEADER":
			info.Leader = value
		case "MYSELF":
			info.Myself = value
			inMyselfSection = true
		case "STATUS":
			if inMyselfSection {
				info.MyRole = strings.ToLower(value)
			}
		case "VERSION":
			if inMyselfSection {
				info.MyVersion = value
			}
		case "NODE-HEALTH":
			if inMyselfSection {
				info.MyHealth = value
			}
		case "NODES":
			for _, n := range strings.Split(value, ",") {
				n = strings.TrimSpace(n)
				if n != "" {
					info.Nodes = append(info.Nodes, n)
				}
			}
		case "REPLICA":
			// Format: host:port | ONLINE | UP-TO-DATE | LOG-SIZE ... | VERSION x.y.z
			parts := strings.Split(value, "|")
			if len(parts) < 2 {
				continue
			}
			rep := raftReplica{
				Host:   strings.TrimSpace(parts[0]),
				Status: strings.TrimSpace(parts[1]),
			}
			if len(parts) >= 3 {
				rep.Sync = strings.TrimSpace(parts[2])
			}
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if strings.HasPrefix(p, "VERSION ") {
					rep.Version = strings.TrimPrefix(p, "VERSION ")
				}
			}
			info.Replicas = append(info.Replicas, rep)
		}
	}

	// Single-node raft clusters may omit the LEADER line entirely.
	// Infer it from MYSELF when STATUS=LEADER so callers don't need to
	// special-case this.
	if info.Leader == "" && info.MyRole == "leader" && info.Myself != "" {
		info.Leader = info.Myself
	}

	return info
}
