package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const persistedUIStateFile = "ui-state.json"

type persistedUIState struct {
	NamespacePath     string `json:"namespace_path"`
	ActiveView        viewID `json:"active_view"`
	CommandLogVisible bool   `json:"command_log_visible"`
}

func defaultPersistedUIState() persistedUIState {
	return persistedUIState{
		ActiveView:        defaultActiveView(),
		CommandLogVisible: true,
	}
}

func loadPersistedUIState() persistedUIState {
	home, err := persistedUIStateHomeDir()
	if err != nil {
		return defaultPersistedUIState()
	}

	path := filepath.Join(home, ".eos-tui", persistedUIStateFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultPersistedUIState()
	}

	var state persistedUIState
	if err := json.Unmarshal(data, &state); err != nil {
		return defaultPersistedUIState()
	}

	state.NamespacePath = cleanPath(state.NamespacePath)
	if state.ActiveView == viewSpaceStatus {
		state.ActiveView = viewSpaces
	}
	if state.ActiveView < 0 || state.ActiveView >= viewCount {
		state.ActiveView = defaultActiveView()
	}
	return state
}

func savePersistedUIState(state persistedUIState) {
	home, err := persistedUIStateHomeDir()
	if err != nil {
		return
	}

	dir := filepath.Join(home, ".eos-tui")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	state.NamespacePath = cleanPath(state.NamespacePath)
	if state.ActiveView == viewSpaceStatus {
		state.ActiveView = viewSpaces
	}
	if state.ActiveView < 0 || state.ActiveView >= viewCount {
		state.ActiveView = defaultActiveView()
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}

	tmp, err := os.CreateTemp(dir, persistedUIStateFile+".*.tmp")
	if err != nil {
		return
	}

	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return
	}
	if err := tmp.Close(); err != nil {
		return
	}

	finalPath := filepath.Join(dir, persistedUIStateFile)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return
	}
	cleanup = false
}

func (m model) persistedUIState() persistedUIState {
	activeView := m.activeView
	if activeView == viewSpaceStatus {
		activeView = viewSpaces
	}
	return persistedUIState{
		NamespacePath:     m.directory.Path,
		ActiveView:        activeView,
		CommandLogVisible: m.commandLog.active,
	}
}

func (m model) persistUIState() {
	savePersistedUIState(m.persistedUIState())
}

func persistedUIStateHomeDir() (string, error) {
	if home := os.Getenv("HOME"); home != "" {
		return home, nil
	}
	return os.UserHomeDir()
}
