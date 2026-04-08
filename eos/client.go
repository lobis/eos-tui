package eos

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	SSHTarget string
	Timeout   time.Duration
}

type Client struct {
	// sshTarget is the gateway/initial SSH host supplied by the user (e.g. "eospilot").
	sshTarget string
	// resolvedSSHTarget is set after MGM master discovery and becomes the
	// effective destination for all subsequent commands.  When empty,
	// sshTarget is used as-is.
	resolvedSSHTarget string
	timeout           time.Duration
}

// effectiveSSHTarget returns the host that runCommand will actually SSH to.
func (c *Client) effectiveSSHTarget() string {
	if c.resolvedSSHTarget != "" {
		return c.resolvedSSHTarget
	}
	return c.sshTarget
}

// ResolvedSSHTarget returns the effective SSH target after master discovery,
// or the original target if discovery has not run.
func (c *Client) ResolvedSSHTarget() string {
	return c.effectiveSSHTarget()
}

// DiscoverMGMMaster runs `redis-cli raft-info` on the current SSH target,
// identifies the QDB/MGM leader, and updates the client so that all subsequent
// commands are routed directly to the leader host.
// Returns the resolved hostname (e.g. "eospilot-ns-02.cern.ch").
func (c *Client) DiscoverMGMMaster(ctx context.Context) (string, error) {
	_ = ctx
	output, err := c.runCommand("redis-cli", "-p", "7777", "raft-info")
	if err != nil {
		return "", fmt.Errorf("raft-info for master discovery: %w", err)
	}

	info := parseRaftInfo(output)
	if info.Leader == "" {
		return "", fmt.Errorf("no leader found in raft-info output")
	}

	leader := hostOnly(info.Leader)
	// EOS nodes run as root; use explicit root@ so the resolved hostname
	// works without relying on SSH config aliases.
	resolved := "root@" + leader
	c.resolvedSSHTarget = resolved
	return resolved, nil
}

type EntryKind string

const (
	EntryKindFile      EntryKind = "file"
	EntryKindContainer EntryKind = "dir"
)

type Entry struct {
	Kind           EntryKind
	Name           string
	Path           string
	ID             uint64
	ParentID       uint64
	Inode          uint64
	UID            uint32
	GID            uint32
	Size           uint64
	TreeSize       int64
	Files          uint64
	Containers     uint64
	LayoutID       uint32
	Flags          uint32
	Mode           uint32
	Locations      int
	LinkName       string
	ETag           string
	ModifiedAt     time.Time
	ChangedAt      time.Time
	SynchronizedAt time.Time
}

type Directory struct {
	Path    string
	Self    Entry
	Entries []Entry
}

type NodeStats struct {
	State       string
	FileCount   uint64
	DirCount    uint64
	BootTime    time.Time
	Uptime      time.Duration
	CurrentFID  uint64
	CurrentCID  uint64
	MemVirtual  uint64
	MemResident uint64
	MemShared   uint64
	MemGrowth   uint64
	ThreadCount uint64
	FileDescs   uint64
}

type FstRecord struct {
	Type            string
	HostPort        string
	Geotag          string
	Status          string
	Activated       string
	HeartbeatDelta  int64
	FileSystemCount int
	EOSVersion      string
	Kernel          string
	Uptime          string
	ThreadCount     uint64
	RSSBytes        uint64
	VSizeBytes      uint64
	CapacityBytes   uint64
	UsedBytes       uint64
	FreeBytes       uint64
	UsedFiles       uint64
	DiskLoad        float64
	ReadRateMB      float64
	WriteRateMB     float64
}

type MgmRecord struct {
	HostPort   string
	Role       string
	Geotag     string
	Status     string
	Heartbeat  string
	EOSVersion string
}

type FileSystemRecord struct {
	Host          string
	Port          uint64
	ID            uint64
	Path          string
	SchedGroup    string
	Geotag        string
	Boot          string
	ConfigStatus  string
	DrainStatus   string
	Active        string
	Health        string
	CapacityBytes uint64
	UsedBytes     uint64
	FreeBytes     uint64
	UsedFiles     uint64
	DiskBWMB      float64
	DiskIOPS      float64
	ReadRateMB    float64
	WriteRateMB   float64
}

type SpaceRecord struct {
	Name          string
	Type          string
	Status        string
	Groups        uint64
	NumFiles      uint64
	NumContainers uint64
	CapacityBytes uint64
	UsedBytes     uint64
	FreeBytes     uint64
}

type SpaceStatusRecord struct {
	Key   string
	Value string
}

type NamespaceStats struct {
	MasterHost              string
	TotalFiles              uint64
	TotalDirectories        uint64
	CurrentFID              uint64
	CurrentCID              uint64
	GeneratedFID            uint64
	GeneratedCID            uint64
	ContentionRead          float64
	ContentionWrite         float64
	CacheFilesMax           uint64
	CacheFilesOccup         uint64
	CacheFilesRequests      uint64
	CacheFilesHits          uint64
	CacheContainersMax      uint64
	CacheContainersOccup    uint64
	CacheContainersRequests uint64
	CacheContainersHits     uint64
}

type IOShapingMode int

const (
	IOShapingApps IOShapingMode = iota
	IOShapingUsers
	IOShapingGroups
)

type IOShapingRecord struct {
	ID        string
	Type      string
	WindowSec int
	ReadBps   float64
	WriteBps  float64
	ReadIOPS  float64
	WriteIOPS float64
}

func New(_ context.Context, cfg Config) (*Client, error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	return &Client{
		sshTarget: cfg.SSHTarget,
		timeout:   timeout,
	}, nil
}

func (c *Client) Close() error {
	return nil
}

func (c *Client) NodeStats(ctx context.Context) (NodeStats, error) {
	_ = ctx
	return c.nodeStatsViaCLI()
}

func (c *Client) MGMs(ctx context.Context) ([]MgmRecord, error) {
	_ = ctx

	// Run redis-cli raft-info directly via runCommand.
	// The SSH target (if set) is always the MGM or an MGM leader node,
	// so we do not need a separate SSH hop.
	output, err := c.runCommand("redis-cli", "-p", "7777", "raft-info")
	if err != nil {
		return nil, fmt.Errorf("redis-cli raft-info: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	info := parseRaftInfo(output)

	if info.Leader == "" && len(info.Nodes) == 0 {
		return nil, fmt.Errorf("no MGM cluster info from raft-info")
	}

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

		mgms = append(mgms, MgmRecord{
			HostPort:   node,
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
		return mgms[i].HostPort < mgms[j].HostPort
	})

	return mgms, nil
}

// raftReplica holds the parsed status of a single replica node from raft-info.
type raftReplica struct {
	Host    string
	Status  string // ONLINE or OFFLINE
	Sync    string // UP-TO-DATE or LAGGING
	Version string
}

// raftInfo holds parsed output of `redis-cli -p 7777 raft-info`.
type raftInfo struct {
	Leader    string
	Myself    string
	MyRole    string
	MyVersion string
	MyHealth  string
	Nodes     []string
	Replicas  []raftReplica
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

	return info
}

// hostOnly strips the port suffix from a host:port string.
func hostOnly(hostPort string) string {
	if idx := strings.LastIndex(hostPort, ":"); idx != -1 {
		return hostPort[:idx]
	}
	return hostPort
}

// HostOnly is the exported version of hostOnly for use by other packages.
func HostOnly(hostPort string) string { return hostOnly(hostPort) }

func (c *Client) Nodes(ctx context.Context) ([]FstRecord, error) {
	_ = ctx

	output, err := c.runCommand("eos", "-j", "node", "ls")
	if err != nil {
		return nil, fmt.Errorf("eos node ls: %w", err)
	}

	var payload struct {
		Result []struct {
			Type           string `json:"type"`
			HostPort       string `json:"hostport"`
			Geotag         string `json:"geotag"`
			Status         string `json:"status"`
			HeartbeatDelta int64  `json:"heartbeatdelta"`
			NoFS           int    `json:"nofs"`
			Cfg            struct {
				Status string `json:"status"`
				Stat   struct {
					Geotag string `json:"geotag"`
					Sys    struct {
						Kernel  string `json:"kernel"`
						RSS     uint64 `json:"rss"`
						Threads uint64 `json:"threads"`
						Uptime  string `json:"uptime"`
						VSize   uint64 `json:"vsize"`
						EOS     struct {
							Version string `json:"version"`
						} `json:"eos"`
					} `json:"sys"`
				} `json:"stat"`
			} `json:"cfg"`
			Avg struct {
				Stat struct {
					Disk struct {
						Load float64 `json:"load"`
					} `json:"disk"`
				} `json:"stat"`
			} `json:"avg"`
			Sum struct {
				Stat struct {
					Disk struct {
						ReadRateMB  float64 `json:"readratemb"`
						WriteRateMB float64 `json:"writeratemb"`
					} `json:"disk"`
					StatFS struct {
						Capacity  uint64 `json:"capacity"`
						FreeBytes uint64 `json:"freebytes"`
						UsedBytes uint64 `json:"usedbytes"`
					} `json:"statfs"`
					UsedFiles uint64 `json:"usedfiles"`
				} `json:"stat"`
			} `json:"sum"`
		} `json:"result"`
	}

	if err := json.Unmarshal(stripEOSPreamble(output), &payload); err != nil {
		return nil, fmt.Errorf("parse node ls: %w (output: %.200s)", err, output)
	}

	nodes := make([]FstRecord, 0, len(payload.Result))
	for _, item := range payload.Result {
		geotag := item.Geotag
		if geotag == "" {
			geotag = item.Cfg.Stat.Geotag
		}

		nodes = append(nodes, FstRecord{
			Type:            item.Type,
			HostPort:        item.HostPort,
			Geotag:          geotag,
			Status:          item.Status,
			Activated:       item.Cfg.Status,
			HeartbeatDelta:  item.HeartbeatDelta,
			FileSystemCount: item.NoFS,
			EOSVersion:      item.Cfg.Stat.Sys.EOS.Version,
			Kernel:          item.Cfg.Stat.Sys.Kernel,
			Uptime:          item.Cfg.Stat.Sys.Uptime,
			ThreadCount:     item.Cfg.Stat.Sys.Threads,
			RSSBytes:        item.Cfg.Stat.Sys.RSS,
			VSizeBytes:      item.Cfg.Stat.Sys.VSize,
			CapacityBytes:   item.Sum.Stat.StatFS.Capacity,
			UsedBytes:       item.Sum.Stat.StatFS.UsedBytes,
			FreeBytes:       item.Sum.Stat.StatFS.FreeBytes,
			UsedFiles:       item.Sum.Stat.UsedFiles,
			DiskLoad:        item.Avg.Stat.Disk.Load,
			ReadRateMB:      item.Sum.Stat.Disk.ReadRateMB,
			WriteRateMB:     item.Sum.Stat.Disk.WriteRateMB,
		})
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].HostPort < nodes[j].HostPort
	})

	return nodes, nil
}

func (c *Client) FileSystems(ctx context.Context) ([]FileSystemRecord, error) {
	_ = ctx

	output, err := c.runCommand("eos", "-j", "-b", "fs", "ls")
	if err != nil {
		return nil, fmt.Errorf("eos fs ls: %w", err)
	}

	var payload struct {
		Result []struct {
			Host         string `json:"host"`
			Port         uint64 `json:"port"`
			ID           uint64 `json:"id"`
			Path         string `json:"path"`
			SchedGroup   string `json:"schedgroup"`
			ConfigStatus string `json:"configstatus"`
			Local        struct {
				Drain struct {
					Status string `json:"status"`
				} `json:"drain"`
			} `json:"local"`
			Stat struct {
				Active string `json:"active"`
				Boot   string `json:"boot"`
				Geotag string `json:"geotag"`
				Health struct {
					Status string `json:"status"`
				} `json:"health"`
				Disk struct {
					BW          float64 `json:"bw"`
					IOPS        float64 `json:"iops"`
					ReadRateMB  float64 `json:"readratemb"`
					WriteRateMB float64 `json:"writeratemb"`
				} `json:"disk"`
				StatFS struct {
					Capacity  uint64 `json:"capacity"`
					FreeBytes uint64 `json:"freebytes"`
					UsedBytes uint64 `json:"usedbytes"`
				} `json:"statfs"`
				UsedFiles uint64 `json:"usedfiles"`
			} `json:"stat"`
		} `json:"result"`
	}

	if err := json.Unmarshal(stripEOSPreamble(output), &payload); err != nil {
		return nil, fmt.Errorf("parse fs ls: %w (output: %.200s)", err, output)
	}

	fileSystems := make([]FileSystemRecord, 0, len(payload.Result))
	for _, item := range payload.Result {
		fileSystems = append(fileSystems, FileSystemRecord{
			Host:          item.Host,
			Port:          item.Port,
			ID:            item.ID,
			Path:          item.Path,
			SchedGroup:    item.SchedGroup,
			Geotag:        item.Stat.Geotag,
			Boot:          item.Stat.Boot,
			ConfigStatus:  item.ConfigStatus,
			DrainStatus:   item.Local.Drain.Status,
			Active:        item.Stat.Active,
			Health:        strings.ReplaceAll(item.Stat.Health.Status, "%20", " "),
			CapacityBytes: item.Stat.StatFS.Capacity,
			UsedBytes:     item.Stat.StatFS.UsedBytes,
			FreeBytes:     item.Stat.StatFS.FreeBytes,
			UsedFiles:     item.Stat.UsedFiles,
			DiskBWMB:      item.Stat.Disk.BW,
			DiskIOPS:      item.Stat.Disk.IOPS,
			ReadRateMB:    item.Stat.Disk.ReadRateMB,
			WriteRateMB:   item.Stat.Disk.WriteRateMB,
		})
	}

	sort.Slice(fileSystems, func(i, j int) bool {
		return fileSystems[i].ID < fileSystems[j].ID
	})

	return fileSystems, nil
}

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

func (c *Client) NamespaceStats(ctx context.Context) (NamespaceStats, error) {
	_ = ctx

	output, err := c.runCommand("eos", "-j", "-b", "ns", "stat")
	if err != nil {
		return NamespaceStats{}, fmt.Errorf("eos ns stat: %w", err)
	}

	var payload struct {
		Result []struct {
			Master string `json:"master_id"`
			NS     struct {
				Total struct {
					Files       any `json:"files"`
					Directories any `json:"directories"`
				} `json:"total"`
				Current struct {
					FID uint64 `json:"fid"`
					CID uint64 `json:"cid"`
				} `json:"current"`
				Generated struct {
					FID uint64 `json:"fid"`
					CID uint64 `json:"cid"`
				} `json:"generated"`
				Contention struct {
					Read  float64 `json:"read"`
					Write float64 `json:"write"`
				} `json:"contention"`
				Cache struct {
					Files struct {
						MaxSize   uint64 `json:"maxsize"`
						Occupancy uint64 `json:"occupancy"`
						Requests  uint64 `json:"requests"`
						Hits      uint64 `json:"hits"`
					} `json:"files"`
					Containers struct {
						MaxSize   uint64 `json:"maxsize"`
						Occupancy uint64 `json:"occupancy"`
						Requests  uint64 `json:"requests"`
						Hits      uint64 `json:"hits"`
					} `json:"containers"`
				} `json:"cache"`
			} `json:"ns"`
		} `json:"result"`
	}

	if err := json.Unmarshal(stripEOSPreamble(output), &payload); err != nil {
		return NamespaceStats{}, fmt.Errorf("parse ns stat: %w (output: %.200s)", err, output)
	}

	stats := NamespaceStats{}
	for _, item := range payload.Result {
		stats.MasterHost = item.Master
		if val := toUint64(item.NS.Total.Files); val > 0 {
			stats.TotalFiles = val
		}
		if val := toUint64(item.NS.Total.Directories); val > 0 {
			stats.TotalDirectories = val
		}
		stats.CurrentFID = item.NS.Current.FID
		stats.CurrentCID = item.NS.Current.CID
		stats.GeneratedFID = item.NS.Generated.FID
		stats.GeneratedCID = item.NS.Generated.CID
		stats.ContentionRead = item.NS.Contention.Read
		stats.ContentionWrite = item.NS.Contention.Write
		stats.CacheFilesMax = item.NS.Cache.Files.MaxSize
		stats.CacheFilesOccup = item.NS.Cache.Files.Occupancy
		stats.CacheFilesRequests = item.NS.Cache.Files.Requests
		stats.CacheFilesHits = item.NS.Cache.Files.Hits
		stats.CacheContainersMax = item.NS.Cache.Containers.MaxSize
		stats.CacheContainersOccup = item.NS.Cache.Containers.Occupancy
		stats.CacheContainersRequests = item.NS.Cache.Containers.Requests
		stats.CacheContainersHits = item.NS.Cache.Containers.Hits
	}

	return stats, nil
}

func (c *Client) ListPath(ctx context.Context, rawPath string) (Directory, error) {
	_ = ctx
	return c.listPathViaCLI(rawPath)
}

func (c *Client) StatPath(ctx context.Context, rawPath string) (Entry, error) {
	_ = ctx
	return c.statPathViaCLI(rawPath)
}

func (c *Client) nodeStatsViaCLI() (NodeStats, error) {
	statusOut, err := c.runCommand("eos", "-b", "status")
	if err != nil {
		return NodeStats{}, fmt.Errorf("eos status: %w", err)
	}

	nsStatOut, err := c.runCommand("eos", "-b", "ns", "stat")
	if err != nil {
		return NodeStats{}, fmt.Errorf("eos ns stat: %w", err)
	}

	stats := NodeStats{
		State: parseStatusHealth(string(statusOut)),
	}

	values := parseLabeledValues(string(nsStatOut))
	stats.FileCount = parseUint(values["Files"])
	stats.DirCount = parseUint(values["Directories"])
	stats.CurrentFID = parseUint(values["current file id"])
	stats.CurrentCID = parseUint(values["current container id"])
	stats.MemVirtual = parseHumanBytes(values["memory virtual"])
	stats.MemResident = parseHumanBytes(values["memory resident"])
	stats.MemShared = parseHumanBytes(values["memory share"])
	stats.MemGrowth = parseHumanBytes(values["memory growths"])
	stats.ThreadCount = parseUint(values["threads"])
	stats.FileDescs = parseUint(values["fds"])
	stats.Uptime = time.Duration(parseUint(values["uptime"])) * time.Second
	if stats.Uptime > 0 {
		stats.BootTime = time.Now().Add(-stats.Uptime)
	}

	return stats, nil
}

// TailLog returns the last n lines of a log file on the effective SSH target
// (or locally when no SSH target is configured).
func (c *Client) TailLog(ctx context.Context, filePath string, n int) ([]byte, error) {
	return c.TailLogOnHost(ctx, "", filePath, n)
}

// TailLogOnHost returns the last n lines of a log file on a specific host.
// When host is empty or matches the current effective target it is equivalent
// to TailLog.  Otherwise the command is routed to the named host, using the
// effective SSH target as a jump proxy when one is configured.
func (c *Client) TailLogOnHost(ctx context.Context, host, filePath string, n int) ([]byte, error) {
	tailArgs := []string{"tail", fmt.Sprintf("-n%d", n), filePath}

	effective := c.effectiveSSHTarget() // e.g. "root@eospilot-ns-02.cern.ch"
	effectiveHost := hostOnly(strings.TrimPrefix(effective, "root@"))

	// Direct case: no specific host, or the host IS the current target.
	if host == "" || host == effectiveHost {
		out, err := c.runCommand(tailArgs...)
		if err != nil {
			return nil, fmt.Errorf("tail %s: %w (output: %.300s)", filePath, err, out)
		}
		return out, nil
	}

	// We need to reach a different host.  Use the effective target as a jump
	// proxy (or SSH directly when running locally).
	target := "root@" + host
	tailCmd := strings.Join(tailArgs, " ")

	c.logCommand(append([]string{"→", target}, tailArgs...))
	ctxTimeout, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var out []byte
	var err error
	if effective == "" {
		out, err = exec.CommandContext(ctxTimeout, "ssh", "-o", "BatchMode=yes", target, tailCmd).CombinedOutput()
	} else {
		out, err = exec.CommandContext(ctxTimeout, "ssh", "-o", "BatchMode=yes", "-J", effective, target, tailCmd).CombinedOutput()
	}
	if err != nil {
		c.logResponse(tailArgs, out, err)
		return nil, fmt.Errorf("tail %s on %s: %w (output: %.300s)", filePath, host, err, out)
	}
	return out, nil
}

// SSHTargetForHost returns the ssh arguments needed to open an interactive
// shell on host, routing via the effective SSH target when necessary.
// Returns (directTarget, jumpProxy) where jumpProxy may be empty.
func (c *Client) SSHTargetForHost(host string) (target, jump string) {
	effective := c.effectiveSSHTarget()
	effectiveHost := hostOnly(strings.TrimPrefix(effective, "root@"))

	if host == "" || host == effectiveHost {
		if effective != "" {
			return effective, ""
		}
		return "", ""
	}

	target = "root@" + host
	if effective != "" {
		return target, effective
	}
	return target, ""
}

func (c *Client) EOSVersion(ctx context.Context) (string, error) {
	_ = ctx
	output, err := c.runCommand("eos", "version")
	if err != nil {
		return "", fmt.Errorf("eos version: %w", err)
	}
	return parseEOSServerVersion(output), nil
}

// parseEOSServerVersion extracts the server version from `eos version` output.
// Example line: "EOS_SERVER_VERSION=5.3.27 EOS_SERVER_RELEASE=unknown"
func parseEOSServerVersion(output []byte) string {
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "EOS_SERVER_VERSION=") {
			rest := strings.TrimPrefix(line, "EOS_SERVER_VERSION=")
			if fields := strings.Fields(rest); len(fields) > 0 {
				return fields[0]
			}
		}
	}
	return ""
}

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

func (c *Client) runCommand(args ...string) ([]byte, error) {
	c.logCommand(args)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	target := c.effectiveSSHTarget()
	var out []byte
	var err error
	if target == "" {
		out, err = exec.CommandContext(ctx, args[0], args[1:]...).CombinedOutput()
	} else {
		remoteCommand := strings.Join(args, " ")
		out, err = exec.CommandContext(ctx, "ssh", "-o", "BatchMode=yes", target, remoteCommand).CombinedOutput()
	}

	if err != nil {
		c.logResponse(args, out, err)
	}
	return out, err
}

func (c *Client) openLogFile() (*os.File, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	logDir := filepath.Join(home, ".eos-tui")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}
	logFile := filepath.Join(logDir, "history.log")
	return os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

func (c *Client) logCommand(args []string) {
	f, err := c.openLogFile()
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	var command string
	if target := c.effectiveSSHTarget(); target != "" {
		// Log as a fully copy-pasteable SSH invocation.
		remoteCmd := strings.Join(args, " ")
		command = fmt.Sprintf("ssh -o BatchMode=yes %s %s", target, shellQuote(remoteCmd))
	} else {
		command = strings.Join(args, " ")
	}
	_, _ = f.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, command))
}

func (c *Client) logResponse(args []string, output []byte, err error) {
	f, ferr := c.openLogFile()
	if ferr != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	// Abbreviate very long output to avoid flooding the log.
	preview := strings.TrimSpace(string(output))
	const maxPreview = 500
	if len(preview) > maxPreview {
		preview = preview[:maxPreview] + "...(truncated)"
	}
	var cmdStr string
	if len(args) > 0 {
		cmdStr = args[len(args)-1] // last arg as a short label
	}
	_, _ = f.WriteString(fmt.Sprintf("[%s] ERROR (%s): %v\n", timestamp, cmdStr, err))
	if preview != "" {
		_, _ = f.WriteString(fmt.Sprintf("[%s]   output: %s\n", timestamp, preview))
	}
}

// stripEOSPreamble removes leading lines that are not part of a JSON payload.
// EOS commands occasionally emit `* <message>` lines on stdout (e.g. error or
// info annotations) before or after the JSON.  This function returns the first
// contiguous block that looks like JSON (starts with `[` or `{`).
func stripEOSPreamble(b []byte) []byte {
	for _, line := range strings.SplitAfter(string(b), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
			// Return from this point to end-of-output.
			idx := strings.Index(string(b), trimmed)
			if idx >= 0 {
				return []byte(strings.TrimSpace(string(b[idx:])))
			}
		}
	}
	return b
}

// shellQuote wraps s in single quotes, escaping any embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func parseStatusHealth(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "health:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "health:"))
		}
	}

	return "-"
}

func toUint64(v any) uint64 {
	switch val := v.(type) {
	case float64:
		return uint64(val)
	case uint64:
		return val
	case int64:
		return uint64(val)
	case int:
		return uint64(val)
	default:
		return 0
	}
}

var nsStatLinePattern = regexp.MustCompile(`^ALL\s+(.+?)\s{2,}(.+)$`)

func parseLabeledValues(output string) map[string]string {
	values := make(map[string]string)
	for _, rawLine := range strings.Split(output, "\n") {
		line := strings.TrimSpace(rawLine)
		matches := nsStatLinePattern.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}

		values[strings.TrimSpace(matches[1])] = strings.TrimSpace(matches[2])
	}

	return values
}

func parseUint(raw string) uint64 {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return 0
	}

	value, _ := strconv.ParseUint(fields[0], 10, 64)
	return value
}

func parseHumanBytes(raw string) uint64 {
	fields := strings.Fields(raw)
	if len(fields) < 2 {
		return 0
	}

	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}

	multiplier := float64(1)
	switch strings.ToUpper(fields[1]) {
	case "KB":
		multiplier = 1024
	case "MB":
		multiplier = 1024 * 1024
	case "GB":
		multiplier = 1024 * 1024 * 1024
	case "TB":
		multiplier = 1024 * 1024 * 1024 * 1024
	}

	return uint64(value * multiplier)
}

type cliFileInfo struct {
	Children       []cliFileInfo `json:"children"`
	CTime          int64         `json:"ctime"`
	CTimeNS        int64         `json:"ctime_ns"`
	ETag           string        `json:"etag"`
	Flags          uint32        `json:"flags"`
	GID            uint32        `json:"gid"`
	ID             uint64        `json:"id"`
	Inode          uint64        `json:"inode"`
	Locations      []cliLocation `json:"locations"`
	Mode           uint32        `json:"mode"`
	MTime          int64         `json:"mtime"`
	MTimeNS        int64         `json:"mtime_ns"`
	Name           string        `json:"name"`
	NFiles         uint64        `json:"nfiles"`
	NNDirectories  uint64        `json:"nndirectories"`
	Path           string        `json:"path"`
	PID            uint64        `json:"pid"`
	Size           uint64        `json:"size"`
	TreeContainers uint64        `json:"treecontainers"`
	TreeFiles      uint64        `json:"treefiles"`
	TreeSize       int64         `json:"treesize"`
	UID            uint32        `json:"uid"`
}

type cliLocation struct {
	FSID uint64 `json:"fsid"`
}

func (c *Client) statPathViaCLI(rawPath string) (Entry, error) {
	info, err := c.fetchCLIFileInfo(rawPath)
	if err != nil {
		return Entry{}, err
	}

	return entryFromCLI(info), nil
}

func (c *Client) listPathViaCLI(rawPath string) (Directory, error) {
	info, err := c.fetchCLIFileInfo(rawPath)
	if err != nil {
		return Directory{}, err
	}

	entries := make([]Entry, 0, len(info.Children))
	for _, child := range info.Children {
		entry := entryFromCLI(child)
		if entry.Path == cleanPath(rawPath) {
			continue
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind == EntryKindContainer
		}

		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	return Directory{
		Path:    cleanPath(rawPath),
		Self:    entryFromCLI(info),
		Entries: entries,
	}, nil
}

func (c *Client) fetchCLIFileInfo(rawPath string) (cliFileInfo, error) {
	output, err := c.runCommand("eos", "-j", "-b", "fileinfo", rawPath)
	if err != nil {
		return cliFileInfo{}, fmt.Errorf("eos fileinfo: %w", err)
	}

	var info cliFileInfo
	if err := json.Unmarshal(stripEOSPreamble(output), &info); err != nil {
		return cliFileInfo{}, fmt.Errorf("parse fileinfo: %w (output: %.200s)", err, output)
	}

	return info, nil
}

func entryFromCLI(info cliFileInfo) Entry {
	fullPath := cleanPath(info.Path)
	name := info.Name
	if name == "" && fullPath == "/" {
		name = "/"
	}

	kind := EntryKindFile
	if info.Mode&040000 != 0 {
		kind = EntryKindContainer
	}

	entry := Entry{
		Kind:       kind,
		Name:       name,
		Path:       fullPath,
		ID:         info.ID,
		ParentID:   info.PID,
		Inode:      info.Inode,
		UID:        info.UID,
		GID:        info.GID,
		Size:       info.Size,
		TreeSize:   info.TreeSize,
		Files:      info.NFiles,
		Containers: info.NNDirectories,
		Flags:      info.Flags,
		Mode:       info.Mode,
		Locations:  len(info.Locations),
		ETag:       info.ETag,
		ModifiedAt: time.Unix(info.MTime, info.MTimeNS).UTC(),
		ChangedAt:  time.Unix(info.CTime, info.CTimeNS).UTC(),
	}

	if entry.Kind == EntryKindContainer {
		entry.Files = info.TreeFiles
		entry.Containers = info.TreeContainers
	}

	return entry
}

func cleanPath(rawPath string) string {
	if rawPath == "" {
		return "/"
	}

	cleaned := path.Clean(rawPath)
	if !strings.HasPrefix(cleaned, "/") {
		return "/" + cleaned
	}

	return cleaned
}
