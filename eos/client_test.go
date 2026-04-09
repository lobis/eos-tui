package eos

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLabeledValues(t *testing.T) {
	input := `
ALL      Files                            78 [booted] (0s)
ALL      Directories                      19
ALL      current file id                  7661
ALL      memory resident                  586.30 MB
`
	values := parseLabeledValues(input)

	if got := parseUint(values["Files"]); got != 78 {
		t.Fatalf("expected files=78, got %d", got)
	}
	if got := parseUint(values["Directories"]); got != 19 {
		t.Fatalf("expected directories=19, got %d", got)
	}
	if got := parseUint(values["current file id"]); got != 7661 {
		t.Fatalf("expected current file id=7661, got %d", got)
	}
	if got := parseHumanBytes(values["memory resident"]); got == 0 {
		t.Fatalf("expected parsed memory resident bytes, got 0")
	}
}

func TestEntryFromCLIContainer(t *testing.T) {
	entry := entryFromCLI(cliFileInfo{
		Name:           "eos",
		Path:           "/eos/",
		ID:             2,
		PID:            1,
		Inode:          2,
		UID:            0,
		GID:            0,
		Mode:           16893,
		TreeFiles:      78,
		TreeContainers: 17,
		TreeSize:       4907360263,
	})

	if entry.Kind != EntryKindContainer {
		t.Fatalf("expected container kind, got %q", entry.Kind)
	}
	if entry.Path != "/eos" {
		t.Fatalf("expected cleaned path /eos, got %q", entry.Path)
	}
	if entry.Files != 78 || entry.Containers != 17 {
		t.Fatalf("unexpected tree counts: files=%d containers=%d", entry.Files, entry.Containers)
	}
}

func TestEntryFromCLIFile(t *testing.T) {
	entry := entryFromCLI(cliFileInfo{
		Name:      "hola",
		Path:      "/eos/dev/test/hola",
		ID:        14,
		PID:       17,
		Inode:     9223372036854775822,
		UID:       0,
		GID:       0,
		Mode:      493,
		Size:      12,
		Locations: []cliLocation{{FSID: 3}},
	})

	if entry.Kind != EntryKindFile {
		t.Fatalf("expected file kind, got %q", entry.Kind)
	}
	if entry.Size != 12 || entry.Locations != 1 {
		t.Fatalf("unexpected file metadata: size=%d locations=%d", entry.Size, entry.Locations)
	}
}

func TestParseSpaceStatus(t *testing.T) {
	input := `
groupbalancer.threshold          := 5
groupmod                         := 24
lru                              := on
tgc.totalbytes                   := 1000000000000000000
`
	records := parseSpaceStatus([]byte(input))

	if len(records) != 4 {
		t.Fatalf("expected 4 records, got %d", len(records))
	}

	if records[0].Key != "groupbalancer.threshold" || records[0].Value != "5" {
		t.Fatalf("unexpected record 0: %+v", records[0])
	}
	if records[3].Key != "tgc.totalbytes" || records[3].Value != "1000000000000000000" {
		t.Fatalf("unexpected record 3: %+v", records[3])
	}
}

func TestNamespaceStatsJSONConflict(t *testing.T) {
	// Simulated JSON with conflicting types for 'files' and 'directories'
	input := `
{
	"result": [
		{
			"ns": {
				"total": {
					"files": 78,
					"directories": 19
				}
			}
		},
		{
			"ns": {
				"total": {
					"files": {
						"changelog": { "size": 0 }
					}
				}
			}
		}
	],
	"retc": "0"
}
`
	var payload struct {
		Result []struct {
			NS struct {
				Total struct {
					Files       any `json:"files"`
					Directories any `json:"directories"`
				} `json:"total"`
			} `json:"ns"`
		} `json:"result"`
	}

	if err := json.Unmarshal([]byte(input), &payload); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(payload.Result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(payload.Result))
	}

	if val := toUint64(payload.Result[0].NS.Total.Files); val != 78 {
		t.Fatalf("expected result[0] files=78, got %v", val)
	}

	if val := toUint64(payload.Result[1].NS.Total.Files); val != 0 {
		t.Fatalf("expected result[1] files=0 (ignored object), got %v", val)
	}
}

func TestParseRaftInfoSingleNode(t *testing.T) {
	// Output from a single-node cluster (lobis-eos-dev / lobisapa-dev.cern.ch).
	input := `TERM 6
LOG-START 0
LOG-SIZE 1710314
LEADER lobisapa-dev.cern.ch:7777
CLUSTER-ID eosdev
COMMIT-INDEX 1710313
LAST-APPLIED 1710313
BLOCKED-WRITES 0
LAST-STATE-CHANGE 2302767 (26 days, 15 hours, 39 minutes, 27 seconds)
----------
MYSELF lobisapa-dev.cern.ch:7777
VERSION 5.4.0.1
STATUS LEADER
NODE-HEALTH GREEN
JOURNAL-FSYNC-POLICY sync-important-updates
----------
MEMBERSHIP-EPOCH 0
NODES lobisapa-dev.cern.ch:7777
OBSERVERS
QUORUM-SIZE 1
`
	info := parseRaftInfo([]byte(input))

	if info.Leader != "lobisapa-dev.cern.ch:7777" {
		t.Errorf("Leader: got %q, want lobisapa-dev.cern.ch:7777", info.Leader)
	}
	if info.Myself != "lobisapa-dev.cern.ch:7777" {
		t.Errorf("Myself: got %q, want lobisapa-dev.cern.ch:7777", info.Myself)
	}
	if info.MyRole != "leader" {
		t.Errorf("MyRole: got %q, want leader", info.MyRole)
	}
	if info.MyVersion != "5.4.0.1" {
		t.Errorf("MyVersion: got %q, want 5.4.0.1", info.MyVersion)
	}
	if info.MyHealth != "GREEN" {
		t.Errorf("MyHealth: got %q, want GREEN", info.MyHealth)
	}
	if len(info.Nodes) != 1 || info.Nodes[0] != "lobisapa-dev.cern.ch:7777" {
		t.Errorf("Nodes: got %v, want [lobisapa-dev.cern.ch:7777]", info.Nodes)
	}
	if len(info.Replicas) != 0 {
		t.Errorf("Replicas: got %d, want 0", len(info.Replicas))
	}
}

func TestParseRaftInfoMultiNode(t *testing.T) {
	// Output from a multi-node cluster (eospilot).
	input := `TERM 214
LOG-START 13282600000
LOG-SIZE 13332650191
LEADER eospilot-ns-02.cern.ch:7777
CLUSTER-ID 9cd69709-1dac-475e-bee6-86e4c9e1f286
COMMIT-INDEX 13332650190
LAST-APPLIED 13332650190
BLOCKED-WRITES 0
LAST-STATE-CHANGE 3431852 (1 months, 9 days, 17 hours, 17 minutes, 32 seconds)
----------
MYSELF eospilot-ns-02.cern.ch:7777
VERSION 5.3.29.1
STATUS LEADER
NODE-HEALTH GREEN
JOURNAL-FSYNC-POLICY sync-important-updates
----------
MEMBERSHIP-EPOCH 13091436169
NODES eospilot-ns-ip700.cern.ch:7777,eospilot-ns-01.cern.ch:7777,eospilot-ns-02.cern.ch:7777
OBSERVERS
QUORUM-SIZE 2
----------
REPLICA eospilot-ns-01.cern.ch:7777 | ONLINE | UP-TO-DATE | LOG-SIZE 13332650191 | VERSION 5.3.29.1
REPLICA eospilot-ns-ip700.cern.ch:7777 | ONLINE | UP-TO-DATE | LOG-SIZE 13332650191 | VERSION 5.3.29.1
`
	info := parseRaftInfo([]byte(input))

	if info.Leader != "eospilot-ns-02.cern.ch:7777" {
		t.Errorf("Leader: got %q, want eospilot-ns-02.cern.ch:7777", info.Leader)
	}
	if info.MyRole != "leader" {
		t.Errorf("MyRole: got %q, want leader", info.MyRole)
	}
	if info.MyVersion != "5.3.29.1" {
		t.Errorf("MyVersion: got %q, want 5.3.29.1", info.MyVersion)
	}
	if len(info.Nodes) != 3 {
		t.Errorf("Nodes count: got %d, want 3", len(info.Nodes))
	}
	if len(info.Replicas) != 2 {
		t.Fatalf("Replicas count: got %d, want 2", len(info.Replicas))
	}

	// Replicas should have the follower nodes (not the leader).
	rep0 := info.Replicas[0]
	if rep0.Host != "eospilot-ns-01.cern.ch:7777" {
		t.Errorf("Replicas[0].Host: got %q", rep0.Host)
	}
	if rep0.Status != "ONLINE" {
		t.Errorf("Replicas[0].Status: got %q, want ONLINE", rep0.Status)
	}
	if rep0.Version != "5.3.29.1" {
		t.Errorf("Replicas[0].Version: got %q, want 5.3.29.1", rep0.Version)
	}
}

func TestParseRaftInfoBuildsMGMRecords(t *testing.T) {
	// Verify that the raftInfo → MgmRecord mapping produces correct roles/versions.
	input := `LEADER eospilot-ns-02.cern.ch:7777
----------
MYSELF eospilot-ns-02.cern.ch:7777
VERSION 5.3.29.1
STATUS LEADER
NODE-HEALTH GREEN
----------
NODES eospilot-ns-ip700.cern.ch:7777,eospilot-ns-01.cern.ch:7777,eospilot-ns-02.cern.ch:7777
----------
REPLICA eospilot-ns-01.cern.ch:7777 | ONLINE | UP-TO-DATE | LOG-SIZE 100 | VERSION 5.3.29.1
REPLICA eospilot-ns-ip700.cern.ch:7777 | OFFLINE | LAGGING | LOG-SIZE 99 | VERSION 5.3.29.0
`
	info := parseRaftInfo([]byte(input))

	leaderHost := hostOnly(info.Leader)
	replicaMap := make(map[string]raftReplica)
	for _, r := range info.Replicas {
		replicaMap[hostOnly(r.Host)] = r
	}

	type want struct {
		role    string
		status  string
		version string
	}
	expectations := map[string]want{
		"eospilot-ns-02.cern.ch":    {role: "leader", status: "online", version: "5.3.29.1"},
		"eospilot-ns-01.cern.ch":    {role: "follower", status: "online", version: "5.3.29.1"},
		"eospilot-ns-ip700.cern.ch": {role: "follower", status: "offline", version: "5.3.29.0"},
	}

	for _, node := range info.Nodes {
		host := hostOnly(node)
		exp, ok := expectations[host]
		if !ok {
			t.Errorf("unexpected node %q", host)
			continue
		}

		role := "follower"
		if host == leaderHost {
			role = "leader"
		}
		if role != exp.role {
			t.Errorf("%s role: got %q, want %q", host, role, exp.role)
		}

		status := "online"
		version := ""
		if r, found := replicaMap[host]; found {
			if strings.ToUpper(r.Status) == "OFFLINE" {
				status = "offline"
			}
			version = r.Version
		}
		if host == leaderHost && info.MyVersion != "" {
			version = info.MyVersion
		}

		if status != exp.status {
			t.Errorf("%s status: got %q, want %q", host, status, exp.status)
		}
		if version != exp.version {
			t.Errorf("%s version: got %q, want %q", host, version, exp.version)
		}
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"eos -j node ls", "'eos -j node ls'"},
		{"it's a test", `'it'\''s a test'`},
		{"simple", "'simple'"},
	}
	for _, tt := range tests {
		if got := shellQuote(tt.input); got != tt.want {
			t.Errorf("shellQuote(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToUint64(t *testing.T) {
	tests := []struct {
		input any
		want  uint64
	}{
		{float64(78), 78},
		{uint64(100), 100},
		{int64(50), 50},
		{int(10), 10},
		{"not a number", 0},
		{map[string]any{"foo": "bar"}, 0},
	}

	for _, tt := range tests {
		if got := toUint64(tt.input); got != tt.want {
			t.Errorf("toUint64(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// ---- stripEOSPreamble tests ------------------------------------------------

func TestStripEOSPreambleNoPreamble(t *testing.T) {
	input := `{"result":[]}`
	got := string(stripEOSPreamble([]byte(input)))
	if got != input {
		t.Errorf("expected no change, got %q", got)
	}
}

func TestStripEOSPreambleArrayNoPreamble(t *testing.T) {
	input := `[{"id":"app1"}]`
	got := string(stripEOSPreamble([]byte(input)))
	if got != input {
		t.Errorf("expected no change for array, got %q", got)
	}
}

func TestStripEOSPreambleSingleStarLine(t *testing.T) {
	input := "* warning: something\n{\"result\":[]}"
	got := string(stripEOSPreamble([]byte(input)))
	if got != `{"result":[]}` {
		t.Errorf("expected preamble stripped, got %q", got)
	}
}

func TestStripEOSPreambleMultipleStarLines(t *testing.T) {
	input := "* info: connecting\n* warning: slow\n[{\"id\":\"x\"}]"
	got := string(stripEOSPreamble([]byte(input)))
	if got != `[{"id":"x"}]` {
		t.Errorf("expected all preamble lines stripped, got %q", got)
	}
}

func TestStripEOSPreamblePreservesTrailingContent(t *testing.T) {
	// A trailing * line after JSON should not truncate JSON.
	input := "{\"result\":[{\"name\":\"default\"}]}"
	got := string(stripEOSPreamble([]byte(input)))
	if !strings.Contains(got, "default") {
		t.Errorf("expected JSON content preserved, got %q", got)
	}
}

func TestStripEOSPreambleEmptyInput(t *testing.T) {
	got := stripEOSPreamble([]byte{})
	if len(got) != 0 {
		t.Errorf("expected empty output for empty input, got %q", got)
	}
}

// ---- JSON parsing with EOS preamble ----------------------------------------

func TestNodesParseWithPreamble(t *testing.T) {
	// Simulate EOS emitting a `* <msg>` line before the JSON payload.
	raw := `* error: cannot connect to localhost
{
  "errormsg": "",
  "result": [
    {
      "type": "nodesview",
      "hostport": "fst01:1095",
      "status": "online",
      "heartbeatdelta": 1,
      "nofs": 5,
      "cfg": {"status": "on", "stat": {"geotag": "local", "sys": {"kernel": "5.x", "rss": 1024, "threads": 50, "uptime": "10h", "vsize": 2048, "eos": {"version": "5.3.27"}}}},
      "avg": {"stat": {"disk": {"load": 0.1}}},
      "sum": {"stat": {"disk": {"readratemb": 1.5, "writeratemb": 2.0}, "statfs": {"capacity": 1000000, "freebytes": 500000, "usedbytes": 500000}, "usedfiles": 100}}
    }
  ]
}`
	var payload struct {
		Result []struct {
			HostPort string `json:"hostport"`
			NoFS     int    `json:"nofs"`
		} `json:"result"`
	}
	if err := json.Unmarshal(stripEOSPreamble([]byte(raw)), &payload); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(payload.Result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(payload.Result))
	}
	if payload.Result[0].HostPort != "fst01:1095" {
		t.Errorf("expected hostport fst01:1095, got %q", payload.Result[0].HostPort)
	}
}

func TestFileSystemsParseWithPreamble(t *testing.T) {
	raw := "* warning: fallback mode\n" + `{
  "errormsg": "",
  "result": [
    {
      "host": "fst01", "port": 1095, "id": 3,
      "path": "/data/fst.1/01",
      "schedgroup": "default.0",
      "configstatus": "rw",
      "local": {"drain": {"status": "nodrain"}},
      "stat": {"active": "online", "boot": "booted", "geotag": "local",
               "health": {"status": "ok"},
               "disk": {"bw": 0.0, "iops": 0.0, "readratemb": 0.0, "writeratemb": 0.0},
               "statfs": {"capacity": 1000, "freebytes": 500, "usedbytes": 500},
               "usedfiles": 10}
    }
  ]
}`
	var payload struct {
		Result []struct {
			ID   uint64 `json:"id"`
			Host string `json:"host"`
			Path string `json:"path"`
		} `json:"result"`
	}
	if err := json.Unmarshal(stripEOSPreamble([]byte(raw)), &payload); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(payload.Result) != 1 || payload.Result[0].Path != "/data/fst.1/01" {
		t.Errorf("unexpected result: %+v", payload.Result)
	}
}

func TestSpacesParseWithPreamble(t *testing.T) {
	raw := "* info: connected\n" + `{
  "errormsg": "",
  "result": [
    {
      "name": "default",
      "type": "groupbalancer",
      "cfg": {"groupsize": 24},
      "sum": {"n_rw": 2, "stat": {"statfs": {"capacity": 1000, "usedbytes": 400, "freebytes": 600, "files": 78}}}
    }
  ]
}`
	var payload struct {
		Result []struct {
			Name string `json:"name"`
		} `json:"result"`
	}
	if err := json.Unmarshal(stripEOSPreamble([]byte(raw)), &payload); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(payload.Result) != 1 || payload.Result[0].Name != "default" {
		t.Errorf("unexpected result: %+v", payload.Result)
	}
}

func TestSessionCommandsReturnsOnlyCommandLines(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "session.log")
	content := strings.Join([]string{
		"[2026-04-09 10:00:00] eos -j node ls",
		"[2026-04-09 10:00:01] ERROR (node ls): boom",
		"[2026-04-09 10:00:01]   output: failed",
		"[2026-04-09 10:00:02] ssh -o BatchMode=yes root@host 'eos -j fs ls'",
	}, "\n") + "\n"
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	client := &Client{sessionLogPath: logPath}
	lines, err := client.SessionCommands(10)
	if err != nil {
		t.Fatalf("SessionCommands error: %v", err)
	}

	if got := strings.Join(lines, "\n"); strings.Contains(got, "ERROR") || strings.Contains(got, "output:") {
		t.Fatalf("expected errors/output lines to be filtered out, got:\n%s", got)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 command lines, got %d", len(lines))
	}
}

func TestSessionCommandsKeepsLastNEntries(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "session.log")
	content := strings.Join([]string{
		"[2026-04-09 10:00:00] cmd-1",
		"[2026-04-09 10:00:01] cmd-2",
		"[2026-04-09 10:00:02] cmd-3",
	}, "\n") + "\n"
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	client := &Client{sessionLogPath: logPath}
	lines, err := client.SessionCommands(2)
	if err != nil {
		t.Fatalf("SessionCommands error: %v", err)
	}
	if got := strings.Join(lines, ","); got != "[2026-04-09 10:00:01] cmd-2,[2026-04-09 10:00:02] cmd-3" {
		t.Fatalf("unexpected tail result: %s", got)
	}
}

func TestNamespaceStatsParseWithPreamble(t *testing.T) {
	raw := "* msg: ns booted\n" + `{
  "errormsg": "",
  "result": [
    {
      "master_id": "mgm01:1094",
      "ns": {
        "total": {"files": 78, "directories": 19},
        "current": {"fid": 7661, "cid": 1000},
        "generated": {"fid": 7662, "cid": 1001},
        "contention": {"read": 0.1, "write": 0.2},
        "cache": {
          "files": {"maxsize": 1000, "occupancy": 500, "requests": 200, "hits": 180},
          "containers": {"maxsize": 500, "occupancy": 250, "requests": 100, "hits": 90}
        }
      }
    }
  ]
}`
	var payload struct {
		Result []struct {
			Master string `json:"master_id"`
			NS     struct {
				Total struct {
					Files any `json:"files"`
				} `json:"total"`
			} `json:"ns"`
		} `json:"result"`
	}
	if err := json.Unmarshal(stripEOSPreamble([]byte(raw)), &payload); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(payload.Result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(payload.Result))
	}
	if payload.Result[0].Master != "mgm01:1094" {
		t.Errorf("expected master mgm01:1094, got %q", payload.Result[0].Master)
	}
	if v := toUint64(payload.Result[0].NS.Total.Files); v != 78 {
		t.Errorf("expected files=78, got %d", v)
	}
}

func TestParseEOSServerVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "standard output",
			input: "EOS_INSTANCE=eosdev\nEOS_SERVER_VERSION=5.3.27 EOS_SERVER_RELEASE=unknown\nEOS_CLIENT_VERSION=5.3.27 EOS_CLIENT_RELEASE=unknown\n",
			want:  "5.3.27",
		},
		{
			name:  "with preamble star line",
			input: "* info: something\nEOS_SERVER_VERSION=5.4.1 EOS_SERVER_RELEASE=1\n",
			want:  "5.4.1",
		},
		{
			name:  "no version line",
			input: "EOS_INSTANCE=eosdev\n",
			want:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEOSServerVersion([]byte(tt.input))
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSSHTargetForHostLocal(t *testing.T) {
	// No SSH target configured (running locally).
	c := &Client{}

	// No host selected — should return empty strings (open local shell).
	target, jump := c.SSHTargetForHost("")
	if target != "" || jump != "" {
		t.Errorf("expected empty target/jump for local+no-host, got target=%q jump=%q", target, jump)
	}

	// Specific host selected — should SSH directly.
	target, jump = c.SSHTargetForHost("fst01.cern.ch")
	if target != "root@fst01.cern.ch" {
		t.Errorf("expected root@fst01.cern.ch, got %q", target)
	}
	if jump != "" {
		t.Errorf("expected no jump for local client, got %q", jump)
	}
}

func TestSSHTargetForHostRemoteSameHost(t *testing.T) {
	// Running via SSH to mgm01.cern.ch; selected host IS the gateway.
	c := &Client{resolvedSSHTarget: "root@mgm01.cern.ch"}

	target, jump := c.SSHTargetForHost("mgm01.cern.ch")
	if target != "root@mgm01.cern.ch" {
		t.Errorf("expected direct target root@mgm01.cern.ch, got %q", target)
	}
	if jump != "" {
		t.Errorf("expected no jump when selected host IS the gateway, got %q", jump)
	}
}

func TestSSHTargetForHostRemoteDifferentHost(t *testing.T) {
	// Running via SSH to mgm01.cern.ch; selected host is a different FST.
	c := &Client{resolvedSSHTarget: "root@mgm01.cern.ch"}

	target, jump := c.SSHTargetForHost("fst01.cern.ch")
	if target != "root@fst01.cern.ch" {
		t.Errorf("expected root@fst01.cern.ch, got %q", target)
	}
	if jump != "root@mgm01.cern.ch" {
		t.Errorf("expected jump root@mgm01.cern.ch, got %q", jump)
	}
}

func TestHostOnly(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"eospilot-ns-02.cern.ch:7777", "eospilot-ns-02.cern.ch"},
		{"lobisapa-dev.cern.ch:7777", "lobisapa-dev.cern.ch"},
		{"hostname", "hostname"},
		{"", ""},
	}
	for _, tt := range tests {
		got := hostOnly(tt.input)
		if got != tt.want {
			t.Errorf("hostOnly(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
