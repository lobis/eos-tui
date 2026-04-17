package eos

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

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

func (c *Client) ListAttrs(ctx context.Context, rawPath string) ([]NamespaceAttr, error) {
	_ = ctx

	output, err := c.runCommand("eos", "attr", "ls", rawPath)
	if err != nil {
		return nil, fmt.Errorf("eos attr ls: %w", err)
	}

	return parseNamespaceAttrs(output), nil
}

func (c *Client) SetAttr(ctx context.Context, rawPath, key, value string, recursive bool) error {
	_ = ctx

	args := []string{"eos", "attr"}
	if recursive {
		args = append(args, "-r")
	}
	args = append(args, "set", fmt.Sprintf("%s=%s", key, value), rawPath)

	_, err := c.runCommand(args...)
	if err != nil {
		return fmt.Errorf("eos attr set: %w", err)
	}
	return nil
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
	fullPath := cleanPath(strings.TrimSpace(info.Path))
	name := strings.TrimSpace(info.Name)
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
		LinkName:   strings.TrimSpace(info.LinkTarget),
		ETag:       strings.TrimSpace(info.ETag),
		ModifiedAt: time.Unix(info.MTime, info.MTimeNS).UTC(),
		ChangedAt:  time.Unix(info.CTime, info.CTimeNS).UTC(),
	}

	if entry.Kind == EntryKindContainer {
		entry.Files = info.TreeFiles
		entry.Containers = info.TreeContainers
	}

	return entry
}

func parseNamespaceAttrs(output []byte) []NamespaceAttr {
	lines := strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n")
	attrs := make([]NamespaceAttr, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "*") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, "\"")
		if key == "" {
			continue
		}

		attrs = append(attrs, NamespaceAttr{
			Key:   key,
			Value: value,
		})
	}

	sort.Slice(attrs, func(i, j int) bool {
		return strings.ToLower(attrs[i].Key) < strings.ToLower(attrs[j].Key)
	})

	return attrs
}
