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
	sshBaseArgs := m.client.SSHArgs(false)
	switch {
	case sshTarget != "" && jumpProxy != "":
		args := append(append([]string{}, sshBaseArgs...), "-t", "-J", jumpProxy, sshTarget)
		cmd = exec.Command("ssh", args...)
	case sshTarget != "":
		args := append(append([]string{}, sshBaseArgs...), "-t", sshTarget)
		cmd = exec.Command("ssh", args...)
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
