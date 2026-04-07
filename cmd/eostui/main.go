package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lobis/eos-tui/internal/eosgrpc"
	"github.com/lobis/eos-tui/internal/ui"
)

func main() {
	var (
		sshTarget   = flag.String("ssh", envOrDefaultCompat([]string{"EOS_TUI_SSH", "EOS_TUI_SSH_TARGET"}, ""), "SSH target for running EOS CLI remotely")
		rootPath    = flag.String("path", envOrDefault("EOS_TUI_PATH", "/"), "initial namespace path")
		timeout     = flag.Duration("timeout", envDurationOrDefault("EOS_TUI_TIMEOUT", 5*time.Second), "per-request timeout")
		noAltScreen = flag.Bool("no-alt-screen", envBoolOrDefault("EOS_TUI_NO_ALT_SCREEN", false), "disable alternate screen mode")
	)
	flag.Parse()

	displayTarget := "local eos cli"
	if *sshTarget != "" {
		displayTarget = fmt.Sprintf("ssh %s", *sshTarget)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := eosgrpc.New(ctx, eosgrpc.Config{
		SSHTarget: *sshTarget,
		Timeout:   *timeout,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "create EOS client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	useAltScreen := !*noAltScreen && terminalSupportsAltScreen()

	options := []tea.ProgramOption{}
	if useAltScreen {
		options = append(options, tea.WithAltScreen())
	}

	program := tea.NewProgram(
		ui.NewModel(client, displayTarget, *rootPath),
		options...,
	)

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "run TUI: %v\n", err)
		os.Exit(1)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
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
