package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lobis/eos-tui/eos"
	"github.com/lobis/eos-tui/ui"
)

func main() {
	var (
		sshTarget         = flag.String("ssh", envOrDefaultCompat([]string{"EOS_TUI_SSH", "EOS_TUI_SSH_TARGET"}, ""), "SSH target for running EOS CLI remotely")
		timeout           = flag.Duration("timeout", envDurationOrDefault("EOS_TUI_TIMEOUT", 15*time.Second), "per-request timeout")
		noAltScreen       = flag.Bool("no-alt-screen", envBoolOrDefault("EOS_TUI_NO_ALT_SCREEN", false), "disable alternate screen mode")
		acceptNewHostKeys = flag.Bool("ssh-accept-new-host-keys", envBoolOrDefault("EOS_TUI_SSH_ACCEPT_NEW_HOST_KEYS", false), "auto-accept first-seen SSH host keys using StrictHostKeyChecking=accept-new")
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Options:")
		fmt.Fprintln(flag.CommandLine.Output(), "  --ssh string")
		fmt.Fprintln(flag.CommandLine.Output(), "        SSH target for running EOS CLI remotely")
		fmt.Fprintln(flag.CommandLine.Output(), "  --timeout duration")
		fmt.Fprintln(flag.CommandLine.Output(), "        per-request timeout")
		fmt.Fprintln(flag.CommandLine.Output(), "  --no-alt-screen")
		fmt.Fprintln(flag.CommandLine.Output(), "        disable alternate screen mode")
		fmt.Fprintln(flag.CommandLine.Output(), "  --ssh-accept-new-host-keys")
		fmt.Fprintln(flag.CommandLine.Output(), "        auto-accept first-seen SSH host keys using StrictHostKeyChecking=accept-new")
	}
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := eos.New(ctx, eos.Config{
		SSHTarget:         *sshTarget,
		Timeout:           *timeout,
		AcceptNewHostKeys: *acceptNewHostKeys,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "create EOS client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	displayTarget := "local eos cli"
	if *sshTarget != "" {
		displayTarget = fmt.Sprintf("ssh %s", *sshTarget)

		// Discover the MGM/QDB master so all subsequent commands go directly
		// to the leader node rather than through the gateway.
		discoverCtx, discoverCancel := context.WithTimeout(context.Background(), *timeout)
		defer discoverCancel()
		if resolved, err := client.DiscoverMGMMaster(discoverCtx); err == nil && resolved != "" {
			displayTarget = fmt.Sprintf("ssh %s  →  %s", *sshTarget, resolved)
		}
	}

	useAltScreen := !*noAltScreen && terminalSupportsAltScreen()

	options := []tea.ProgramOption{}
	if useAltScreen {
		options = append(options, tea.WithAltScreen())
	}

	program := tea.NewProgram(
		ui.NewModel(client, displayTarget, ""),
		options...,
	)

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "run TUI: %v\n", err)
		os.Exit(1)
	}
}

func envOrDefaultCompat(keys []string, fallback string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}

	return fallback
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func envBoolOrDefault(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func terminalSupportsAltScreen() bool {
	term := strings.TrimSpace(strings.ToLower(os.Getenv("TERM")))
	return term != "" && term != "dumb"
}
