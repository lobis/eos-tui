package ui

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) openShell() (tea.Model, tea.Cmd) {
	if m.client == nil {
		return m, nil
	}

	selectedHost := m.selectedHostForView()
	if selectedHost == "" {
		return m, nil
	}

	sshTarget, jumpProxy := m.client.SSHTargetForHost(selectedHost)

	var cmd *exec.Cmd
	switch {
	case sshTarget != "" && jumpProxy != "":
		cmd = exec.Command("ssh", "-o", "BatchMode=no", "-t", "-J", jumpProxy, sshTarget)
	case sshTarget != "":
		cmd = exec.Command("ssh", "-o", "BatchMode=no", "-t", sshTarget)
	default:
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/bash"
		}
		cmd = exec.Command(shell)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			m.status = fmt.Sprintf("shell exited: %v", err)
		}
		return tea.ClearScreen
	})
}
