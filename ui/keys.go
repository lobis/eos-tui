package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lobis/eos-tui/eos"
)

func (m model) updateFSTKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	fsts := m.visibleFSTs()
	half := max(1, m.height/6)
	switch msg.String() {
	case "/":
		m.fstFilter.column = m.fstColumnSelected
		m.openFilterPopup()
		return m, nil
	case "left":
		m.fstColumnSelected = max(0, m.fstColumnSelected-1)
		m.status = fmt.Sprintf("Selected node column: %s", m.fstSelectedColumnLabel())
	case "right":
		m.fstColumnSelected = min(nodeColumnCount()-1, m.fstColumnSelected+1)
		m.status = fmt.Sprintf("Selected node column: %s", m.fstSelectedColumnLabel())
	case "S":
		m.fstSort = m.nextNodeSortState()
		m.fstSelected = clampIndex(0, len(m.visibleFSTs()))
		m.status = fmt.Sprintf("Node sort: %s", m.fstSortStateLabel())
	case "c":
		delete(m.fstFilter.filters, m.fstColumnSelected)
		m.fstFilter.column = m.fstColumnSelected
		m.status = fmt.Sprintf("Cleared node filter on %s", m.fstSelectedColumnLabel())
	case "up", "k":
		if m.fstSelected > 0 {
			m.fstSelected--
		}
	case "down", "j":
		if m.fstSelected < len(fsts)-1 {
			m.fstSelected++
		}
	case "ctrl+u":
		m.fstSelected = max(0, m.fstSelected-half)
	case "ctrl+d":
		m.fstSelected = min(len(fsts)-1, m.fstSelected+half)
	case "g":
		m.fstSelected = 0
	case "G":
		m.fstSelected = max(0, len(fsts)-1)
	}

	return m, nil
}

func (m model) updateMGMKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.mgms)
	half := max(1, m.height/6)
	switch msg.String() {
	case "up", "k":
		if m.mgmSelected > 0 {
			m.mgmSelected--
		}
	case "down", "j":
		if m.mgmSelected < n-1 {
			m.mgmSelected++
		}
	case "ctrl+u":
		m.mgmSelected = max(0, m.mgmSelected-half)
	case "ctrl+d":
		m.mgmSelected = min(n-1, m.mgmSelected+half)
	case "g":
		m.mgmSelected = 0
	case "G":
		m.mgmSelected = max(0, n-1)
	}
	return m, nil
}

func (m model) updateQDBKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.mgms)
	half := max(1, m.height/6)
	switch msg.String() {
	case "up", "k":
		if m.qdbSelected > 0 {
			m.qdbSelected--
		}
	case "down", "j":
		if m.qdbSelected < n-1 {
			m.qdbSelected++
		}
	case "ctrl+u":
		m.qdbSelected = max(0, m.qdbSelected-half)
	case "ctrl+d":
		m.qdbSelected = min(n-1, m.qdbSelected+half)
	case "g":
		m.qdbSelected = 0
	case "G":
		m.qdbSelected = max(0, n-1)
	}
	return m, nil
}

func (m model) updateFileSystemKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	fileSystems := m.visibleFileSystems()
	half := max(1, m.height/6)
	switch msg.String() {
	case "/":
		m.fsFilter.column = m.fsColumnSelected
		m.openFilterPopup()
		return m, nil
	case "enter":
		return m.openFSConfigStatusEdit()
	case "left":
		m.fsColumnSelected = max(0, m.fsColumnSelected-1)
		m.status = fmt.Sprintf("Selected filesystem column: %s", m.fsSelectedColumnLabel())
	case "right":
		m.fsColumnSelected = min(fsColumnCount()-1, m.fsColumnSelected+1)
		m.status = fmt.Sprintf("Selected filesystem column: %s", m.fsSelectedColumnLabel())
	case "S":
		m.fsSort = m.nextFileSystemSortState()
		m.fsSelected = clampIndex(0, len(m.visibleFileSystems()))
		m.status = fmt.Sprintf("Filesystem sort: %s", m.fsSortStateLabel())
	case "c":
		delete(m.fsFilter.filters, m.fsColumnSelected)
		m.fsFilter.column = m.fsColumnSelected
		m.status = fmt.Sprintf("Cleared filesystem filter on %s", m.fsSelectedColumnLabel())
	case "up", "k":
		if m.fsSelected > 0 {
			m.fsSelected--
		}
	case "down", "j":
		if m.fsSelected < len(fileSystems)-1 {
			m.fsSelected++
		}
	case "ctrl+u":
		m.fsSelected = max(0, m.fsSelected-half)
	case "ctrl+d":
		m.fsSelected = min(len(fileSystems)-1, m.fsSelected+half)
	case "g":
		m.fsSelected = 0
	case "G":
		m.fsSelected = max(0, len(fileSystems)-1)
	}

	return m, nil
}

func (m model) updateNamespaceKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	half := max(1, m.height/6)
	selectionChanged := false
	switch msg.String() {
	case "up", "k":
		if m.nsSelected > 0 {
			m.nsSelected--
			selectionChanged = true
		}
	case "down", "j":
		if m.nsSelected < len(m.directory.Entries)-1 {
			m.nsSelected++
			selectionChanged = true
		}
	case "ctrl+u":
		m.nsSelected = max(0, m.nsSelected-half)
		selectionChanged = true
	case "ctrl+d":
		m.nsSelected = min(len(m.directory.Entries)-1, m.nsSelected+half)
		selectionChanged = true
	case "G":
		m.nsSelected = max(0, len(m.directory.Entries)-1)
		selectionChanged = true
	case "g":
		m.nsSelected = 0
		selectionChanged = true
	case "backspace", "left":
		parent := parentPath(m.directory.Path)
		if parent != m.directory.Path {
			m.nsSelected = 0
			m.nsLoading = true
			m.status = fmt.Sprintf("Opening %s...", parent)
			return m, loadDirectoryCmd(m.client, parent)
		}
	case "enter":
		return m.startNamespaceAttrEdit()
	case "right":
		entry, ok := m.selectedNamespaceEntry()
		if ok && entry.Kind == eos.EntryKindContainer {
			m.nsSelected = 0
			m.nsLoading = true
			m.status = fmt.Sprintf("Opening %s...", entry.Path)
			return m, loadDirectoryCmd(m.client, entry.Path)
		}
	}

	if selectionChanged {
		m = m.rememberNamespaceDetailContent()
		return m.startNamespaceAttrLoad(false)
	}

	return m, nil
}

func (m model) startNamespaceAttrEdit() (tea.Model, tea.Cmd) {
	targetPath := m.currentNamespaceAttrTargetPath()
	if m.nsAttrsLoading && m.nsAttrsTargetPath == targetPath {
		m.status = "Attributes are still loading..."
		return m, nil
	}
	if m.nsAttrsErr != nil && m.nsAttrsTargetPath == targetPath {
		m.status = "Cannot edit attributes until they load successfully"
		return m, nil
	}
	if !m.nsAttrsLoaded || m.nsAttrsTargetPath != targetPath || len(m.nsAttrs) == 0 {
		m.status = "No attributes available to edit"
		return m, nil
	}

	input := textinput.New()
	input.Prompt = "value> "
	input.CharLimit = 4096
	input.Width = 48

	m.nsAttrEdit = namespaceAttrEdit{
		active:     true,
		stage:      attrEditStageSelect,
		targetPath: targetPath,
		attrs:      append([]eos.NamespaceAttr(nil), m.nsAttrs...),
		selected:   0,
		input:      input,
	}
	return m, nil
}

func (m model) updateNamespaceAttrEditKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.nsAttrEdit.stage {
	case attrEditStageSelect:
		switch msg.String() {
		case "esc":
			m.nsAttrEdit.active = false
			return m, nil
		case "g":
			m.nsAttrEdit.selected = 0
		case "G":
			m.nsAttrEdit.selected = max(0, len(m.nsAttrEdit.attrs)-1)
		case "up", "k":
			if m.nsAttrEdit.selected > 0 {
				m.nsAttrEdit.selected--
			}
		case "down", "j":
			if m.nsAttrEdit.selected < len(m.nsAttrEdit.attrs)-1 {
				m.nsAttrEdit.selected++
			}
		case "enter":
			attr := m.nsAttrEdit.attrs[m.nsAttrEdit.selected]
			m.nsAttrEdit.stage = attrEditStageInput
			m.nsAttrEdit.input.SetValue(attr.Value)
			return m, m.nsAttrEdit.input.Focus()
		}
		return m, nil
	case attrEditStageInput:
		switch msg.String() {
		case "esc":
			m.nsAttrEdit.active = false
			return m, nil
		case "enter":
			attr := m.nsAttrEdit.attrs[m.nsAttrEdit.selected]
			m.nsAttrEdit.active = false
			return m, runNamespaceAttrSetCmd(m.client, m.nsAttrEdit.targetPath, attr.Key, m.nsAttrEdit.input.Value())
		}

		var cmd tea.Cmd
		m.nsAttrEdit.input, cmd = m.nsAttrEdit.input.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

func (m model) updateSpacesKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	half := max(1, m.height/6)
	switch msg.String() {
	case "up", "k":
		if m.spacesSelected > 0 {
			m.spacesSelected--
		}
	case "down", "j":
		if m.spacesSelected < len(m.spaces)-1 {
			m.spacesSelected++
		}
	case "ctrl+u":
		m.spacesSelected = max(0, m.spacesSelected-half)
	case "ctrl+d":
		m.spacesSelected = min(len(m.spaces)-1, m.spacesSelected+half)
	case "g":
		m.spacesSelected = 0
	case "G":
		m.spacesSelected = max(0, len(m.spaces)-1)
	case "left":
		m.spacesColumnSelected = max(0, m.spacesColumnSelected-1)
	case "right":
		m.spacesColumnSelected = min(6, m.spacesColumnSelected+1)
	}

	return m, nil
}

func (m model) updateSpaceStatusKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	half := max(1, m.height/6)
	switch msg.String() {
	case "up", "k":
		if m.spaceStatusSelected > 0 {
			m.spaceStatusSelected--
		}
	case "down", "j":
		if m.spaceStatusSelected < len(m.spaceStatus)-1 {
			m.spaceStatusSelected++
		}
	case "ctrl+u":
		m.spaceStatusSelected = max(0, m.spaceStatusSelected-half)
	case "ctrl+d":
		m.spaceStatusSelected = min(len(m.spaceStatus)-1, m.spaceStatusSelected+half)
	case "g":
		m.spaceStatusSelected = 0
	case "G":
		m.spaceStatusSelected = max(0, len(m.spaceStatus)-1)
	}

	return m, nil
}

func (m model) updateIOShapingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	half := max(1, m.height/6)
	n := len(m.ioShapingMergedRows())
	switch msg.String() {
	case "a":
		if m.ioShapingMode != eos.IOShapingApps {
			m.ioShapingMode = eos.IOShapingApps
			m.ioShapingSelected = 0
			m.ioShapingLoading = true
			return m, tea.Batch(loadIOShapingCmd(m.client, m.ioShapingMode), loadIOShapingPoliciesCmd(m.client))
		}
	case "u":
		if m.ioShapingMode != eos.IOShapingUsers {
			m.ioShapingMode = eos.IOShapingUsers
			m.ioShapingSelected = 0
			m.ioShapingLoading = true
			return m, tea.Batch(loadIOShapingCmd(m.client, m.ioShapingMode), loadIOShapingPoliciesCmd(m.client))
		}
	case "g":
		if m.ioShapingMode != eos.IOShapingGroups {
			m.ioShapingMode = eos.IOShapingGroups
			m.ioShapingSelected = 0
			m.ioShapingLoading = true
			return m, tea.Batch(loadIOShapingCmd(m.client, m.ioShapingMode), loadIOShapingPoliciesCmd(m.client))
		}
	case "up", "k":
		if m.ioShapingSelected > 0 {
			m.ioShapingSelected--
		}
	case "down", "j":
		if m.ioShapingSelected < n-1 {
			m.ioShapingSelected++
		}
	case "ctrl+u":
		m.ioShapingSelected = max(0, m.ioShapingSelected-half)
	case "ctrl+d":
		m.ioShapingSelected = min(n-1, m.ioShapingSelected+half)
	case "G":
		m.ioShapingSelected = max(0, n-1)
	case "enter":
		return m.startIOShapingPolicyEdit()
	case "d":
		return m.startIOShapingPolicyDeleteConfirm()
	}
	m.ioShapingSelected = clampIndex(m.ioShapingSelected, n)
	return m, nil
}

func (m model) updateGroupKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	groups := m.visibleGroups()
	switch msg.String() {
	case "up", "k":
		m.groupsSelected = max(0, m.groupsSelected-1)
	case "down", "j":
		m.groupsSelected = min(len(groups)-1, m.groupsSelected+1)
	case "ctrl+d", "pgdown":
		m.groupsSelected = min(len(groups)-1, m.groupsSelected+10)
	case "ctrl+u", "pgup":
		m.groupsSelected = max(0, m.groupsSelected-10)
	case "left", "h":
		m.groupsColumnSelected = max(0, m.groupsColumnSelected-1)
	case "right", "l":
		m.groupsColumnSelected = min(groupColumnCount()-1, m.groupsColumnSelected+1)
	case "S":
		sortCol := groupSortColumn(m.groupsColumnSelected)
		if m.groupSort.column == int(sortCol) {
			m.groupSort.desc = !m.groupSort.desc
		} else {
			m.groupSort.column = int(sortCol)
			m.groupSort.desc = false
		}
	case "/":
		m.openFilterPopup()
	}
	return m, nil
}

func (m model) updateSpaceStatusEditKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.edit.active = false
		return m, nil
	case "g":
		if !m.edit.focusInput {
			m.edit.button = buttonCancel
			return m, nil
		}
	case "G":
		if !m.edit.focusInput {
			m.edit.button = buttonContinue
			return m, nil
		}
	case "tab", "shift+tab":
		m.edit.focusInput = !m.edit.focusInput
		if m.edit.focusInput {
			return m, m.edit.input.Focus()
		} else {
			m.edit.input.Blur()
			return m, nil
		}
	case "up", "down":
		if m.edit.stage == editStageInput {
			m.edit.focusInput = !m.edit.focusInput
			if m.edit.focusInput {
				return m, m.edit.input.Focus()
			} else {
				m.edit.input.Blur()
				return m, nil
			}
		}
	case "left", "right":
		if !m.edit.focusInput {
			if m.edit.button == buttonCancel {
				m.edit.button = buttonContinue
			} else {
				m.edit.button = buttonCancel
			}
		}
	case "enter":
		if !m.edit.focusInput && m.edit.button == buttonCancel {
			m.edit.active = false
			return m, nil
		}

		if m.edit.stage == editStageInput {
			if m.edit.focusInput || m.edit.button == buttonContinue {
				m.edit.stage = editStageConfirm
				m.edit.button = buttonCancel
				m.edit.focusInput = false
				m.edit.input.Blur()
				return m, nil
			}
		} else if m.edit.stage == editStageConfirm {
			if m.edit.button == buttonContinue {
				return m, runSpaceConfigCmd(m.client, m.edit.record.Key, m.edit.input.Value())
			}
		}
	}

	if m.edit.focusInput {
		var cmd tea.Cmd
		m.edit.input, cmd = m.edit.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) updateFSConfigStatusEditKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.fsEdit.active = false
		return m, nil
	case "g":
		m.fsEdit.selected = 0
	case "G":
		m.fsEdit.selected = len(configStatusOptions) - 1
	case "up", "k":
		if m.fsEdit.selected > 0 {
			m.fsEdit.selected--
		}
	case "down", "j":
		if m.fsEdit.selected < len(configStatusOptions)-1 {
			m.fsEdit.selected++
		}
	case "enter":
		chosen := configStatusOptions[m.fsEdit.selected]
		fsID := m.fsEdit.fsID
		m.fsEdit.active = false
		return m, runFsConfigStatusCmd(m.client, fsID, chosen)
	}
	return m, nil
}

func (m model) updatePopup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeFilterPopup("Filter selection cancelled")
		return m, nil
	case "enter":
		m.applyPopupSelection()
		return m, nil
	case "up", "down", "j", "k", "pgup", "pgdown", "home", "end", "b", "f", "u", "d", "g", "G":
		var cmd tea.Cmd
		m.popup.table, cmd = m.popup.table.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.popup.input, cmd = m.popup.input.Update(msg)
	m.updatePopupRows()
	return m, cmd
}

func (m model) updateLogKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Filter input mode: type to grep, enter to apply, esc to cancel.
	if m.log.filtering {
		switch msg.String() {
		case "esc":
			m.log.filtering = false
			m.log.input.Blur()
		case "enter":
			m.log.filtering = false
			m.log.input.Blur()
			m.log.filter = m.log.input.Value()
			m.log.filtered = applyLogFilter(m.log.allLines, m.log.filter)
			m.log.vp.SetContent(strings.Join(m.log.filtered, "\n"))
			m.log.vp.GotoBottom()
		default:
			var cmd tea.Cmd
			m.log.input, cmd = m.log.input.Update(msg)
			// Live filter as user types.
			m.log.filtered = applyLogFilter(m.log.allLines, m.log.input.Value())
			m.log.vp.SetContent(strings.Join(m.log.filtered, "\n"))
			m.log.vp.GotoBottom()
			return m, cmd
		}
		return m, nil
	}

	// Normal navigation.
	switch msg.String() {
	case "esc", "q":
		m.log = logOverlay{}
	case "/":
		m.log.filtering = true
		m.log.input.SetValue(m.log.filter)
		m.log.input.Focus()
	case "r":
		m.log.loading = true
		return m, loadLogCmd(m.client, m.log.host, m.log.filePath)
	case "g":
		m.log.vp.GotoTop()
	case "G":
		m.log.vp.GotoBottom()
	default:
		var cmd tea.Cmd
		m.log.vp, cmd = m.log.vp.Update(msg)
		return m, cmd
	}
	return m, nil
}
