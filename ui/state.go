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

func loadPersistedUIState() persistedUIState {
	home, err := os.UserHomeDir()
	if err != nil {
		return persistedUIState{}
	}

	path := filepath.Join(home, ".eos-tui", persistedUIStateFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return persistedUIState{}
	}

	var state persistedUIState
	if err := json.Unmarshal(data, &state); err != nil {
		return persistedUIState{}
	}

	state.NamespacePath = cleanPath(state.NamespacePath)
	if state.ActiveView < 0 || state.ActiveView >= viewCount {
		state.ActiveView = viewMGM
	}
	return state
}

func savePersistedUIState(state persistedUIState) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	dir := filepath.Join(home, ".eos-tui")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	state.NamespacePath = cleanPath(state.NamespacePath)
	if state.ActiveView < 0 || state.ActiveView >= viewCount {
		state.ActiveView = viewMGM
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
	return persistedUIState{
		NamespacePath:     m.directory.Path,
		ActiveView:        m.activeView,
		CommandLogVisible: m.commandLog.active,
	}
}

func (m model) persistUIState() {
	savePersistedUIState(m.persistedUIState())
}
