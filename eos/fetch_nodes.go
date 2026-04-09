package eos

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

func (c *Client) NodeStats(ctx context.Context) (NodeStats, error) {
	_ = ctx
	return c.nodeStatsViaCLI()
}

func (c *Client) nodeStatsViaCLI() (NodeStats, error) {
	// Fetch namespace stats via eos ns stat (plain text format).
	// State (health) is not fetched here; it is derived in the UI from the
	// already-loaded node and filesystem data, avoiding a redundant call to
	// `eos status` which internally runs the eos-status shell script and
	// creates temporary files under /tmp.
	nsStatOut, err := c.runCommand("eos", "-b", "ns", "stat")
	if err != nil {
		return NodeStats{}, fmt.Errorf("eos ns stat: %w", err)
	}

	stats := NodeStats{}

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
		h, p := splitHostPort(item.HostPort)
		nodes = append(nodes, FstRecord{
			Type:            item.Type,
			Host:            h,
			Port:            p,
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
		if nodes[i].Host != nodes[j].Host {
			return nodes[i].Host < nodes[j].Host
		}
		return nodes[i].Port < nodes[j].Port
	})

	return nodes, nil
}
