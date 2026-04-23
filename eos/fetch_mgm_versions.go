package eos

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

func (c *Client) MGMVersions(ctx context.Context, mgms []MgmRecord) (map[string]string, map[string]string, error) {
	mgmHosts := uniqueHosts(func(record MgmRecord) string { return record.Host }, mgms)
	qdbHosts := uniqueHosts(func(record MgmRecord) string { return record.QDBHost }, mgms)

	mgmVersions := make(map[string]string, len(mgmHosts))
	qdbVersions := make(map[string]string, len(qdbHosts))

	type versionResult struct {
		host    string
		version string
		kind    string
		err     error
	}

	results := make(chan versionResult, len(mgmHosts)+len(qdbHosts))
	var wg sync.WaitGroup

	for _, host := range mgmHosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			version, err := c.eosVersionOnHost(ctx, host)
			results <- versionResult{host: host, version: version, kind: "mgm", err: err}
		}(host)
	}

	for _, host := range qdbHosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			version, err := c.qdbVersionOnHost(ctx, host)
			results <- versionResult{host: host, version: version, kind: "qdb", err: err}
		}(host)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var errs []string
	for result := range results {
		if result.version != "" {
			switch result.kind {
			case "mgm":
				mgmVersions[result.host] = result.version
			case "qdb":
				qdbVersions[result.host] = result.version
			}
		}
		if result.err != nil {
			errs = append(errs, fmt.Sprintf("%s %s: %v", result.kind, result.host, result.err))
		}
	}

	if len(errs) > 0 {
		return mgmVersions, qdbVersions, errors.New(strings.Join(errs, "; "))
	}

	return mgmVersions, qdbVersions, nil
}

func uniqueHosts(selector func(MgmRecord) string, mgms []MgmRecord) []string {
	seen := make(map[string]struct{}, len(mgms))
	hosts := make([]string, 0, len(mgms))
	for _, record := range mgms {
		host := strings.TrimSpace(selector(record))
		if host == "" {
			continue
		}
		if _, ok := seen[host]; ok {
			continue
		}
		seen[host] = struct{}{}
		hosts = append(hosts, host)
	}
	return hosts
}

func (c *Client) eosVersionOnHost(ctx context.Context, host string) (string, error) {
	output, err := c.runCommandOnHost(ctx, host, "eos", "--version")
	if err != nil {
		return "", fmt.Errorf("eos --version: %w", err)
	}
	version := parseEOSServerVersion(output)
	if version == "" {
		return "", fmt.Errorf("eos --version: no EOS_SERVER_VERSION found")
	}
	return version, nil
}

func (c *Client) qdbVersionOnHost(ctx context.Context, host string) (string, error) {
	output, err := c.runCommandOnHost(ctx, host, "redis-cli", "-p", "7777", "raft-info")
	if err != nil {
		return "", fmt.Errorf("redis-cli raft-info: %w", err)
	}
	info := parseRaftInfo(output)
	if info.MyVersion != "" {
		return info.MyVersion, nil
	}
	for _, replica := range info.Replicas {
		if hostOnly(replica.Host) == host && replica.Version != "" {
			return replica.Version, nil
		}
	}
	return "", fmt.Errorf("redis-cli raft-info: no VERSION found")
}
