package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lobis/eos-tui/eos"
)

func NewModel(client *eos.Client, endpoint, rootPath string) tea.Model {
	input := textinput.New()
	input.Prompt = "filter> "
	input.CharLimit = 256
	input.Width = 40
	input.Focus()

	popupTable := table.New(
		table.WithColumns([]table.Column{{Title: "value", Width: 40}}),
		table.WithRows(nil),
		table.WithFocused(true),
		table.WithHeight(8),
	)

	return model{
		client:             client,
		endpoint:           endpoint,
		width:              120,
		height:             32,
		activeView:         viewMGM,
		fstStatsLoading:    true,
		fstsLoading:        true,
		fileSystemsLoading: true,
		spacesLoading:      true,
		nsStatsLoading:     true,
		nsLoading:          false,
		spaceStatusLoading: true,
		directory: eos.Directory{
			Path: cleanPath(rootPath),
		},
		status:               "Loading EOS state...",
		fstColumnSelected:    int(fstFilterHost),
		fsColumnSelected:     int(fsFilterHost),
		groupsColumnSelected: int(groupFilterName),
		fstSort:              sortState{column: int(fstSortNone)},
		fsSort:               sortState{column: int(fsSortNone)},
		groupSort:            sortState{column: int(groupSortNone)},
		fstFilter:            filterState{filters: map[int]string{}},
		fsFilter:             filterState{filters: map[int]string{}},
		groupFilter:          filterState{filters: map[int]string{}},
		popup: filterPopup{
			input: input,
			table: popupTable,
		},
		styles: newStyles(),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadInfraCmd(m.client), tickCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
		if msg.Height > 0 {
			m.height = msg.Height
		}
		return m, tea.ClearScreen
	case tea.KeyMsg:
		// Log overlay intercepts all keys when active.
		if m.log.active {
			return m.updateLogKeys(msg)
		}
		if m.alert.active {
			if msg.String() == "enter" || msg.String() == "esc" {
				m.alert.active = false
			}
			return m, nil
		}
		if m.popup.active {
			return m.updatePopup(msg)
		}
		if m.edit.active {
			return m.updateSpaceStatusEditKeys(msg)
		}
		if m.fsEdit.active {
			return m.updateFSConfigStatusEditKeys(msg)
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			switch m.activeView {
			case viewFST:
				if len(m.fstFilter.filters) > 0 {
					m.fstFilter.filters = map[int]string{}
					m.status = "Node filters cleared"
				}
			case viewFileSystems:
				if len(m.fsFilter.filters) > 0 {
					m.fsFilter.filters = map[int]string{}
					m.status = "Filesystem filters cleared"
				}
			}
			return m, nil
		case "tab":
			m.activeView = (m.activeView + 1) % viewCount
			return m.onViewChanged()
		case "shift+tab":
			m.activeView = (m.activeView + viewCount - 1) % viewCount
			return m.onViewChanged()
		case "1":
			m.activeView = viewMGM
			return m.onViewChanged()
		case "2":
			m.activeView = viewQDB
			return m.onViewChanged()
		case "3":
			m.activeView = viewFST
			return m.onViewChanged()
		case "4":
			m.activeView = viewFileSystems
			return m.onViewChanged()
		case "5":
			m.activeView = viewNamespace
			return m.onViewChanged()
		case "6":
			m.activeView = viewSpaces
			return m.onViewChanged()
		case "7":
			m.activeView = viewNamespaceStats
			return m.onViewChanged()
		case "8":
			m.activeView = viewSpaceStatus
			return m.onViewChanged()
		case "9":
			m.activeView = viewIOShaping
			return m.onViewChanged()
		case "0":
			m.activeView = viewGroups
			return m.onViewChanged()
		case "r":
			return m.refreshActiveView()
		case "l":
			return m.openLogOverlay()
		case "s":
			return m.openShell()
		}

		switch m.activeView {
		case viewMGM:
			return m.updateMGMKeys(msg)
		case viewQDB:
			return m.updateQDBKeys(msg)
		case viewFST:
			return m.updateFSTKeys(msg)
		case viewFileSystems:
			return m.updateFileSystemKeys(msg)
		case viewNamespace:
			return m.updateNamespaceKeys(msg)
		case viewSpaces:
			return m.updateSpacesKeys(msg)
		case viewNamespaceStats:
			// read-only
		case viewSpaceStatus:
			if msg.String() == "enter" {
				return m.startSpaceStatusEdit()
			}
			return m.updateSpaceStatusKeys(msg)
		case viewIOShaping:
			return m.updateIOShapingKeys(msg)
		case viewGroups:
			return m.updateGroupKeys(msg)
		}
	case mgmsLoadedMsg:
		m.mgmsLoading = false
		m.mgms = msg.mgms
		m.mgmsErr = msg.err
		return m, nil

	case infraLoadedMsg:
		m.fstStatsLoading = false
		m.fstsLoading = false
		m.mgmsLoading = false
		m.fileSystemsLoading = false
		if msg.eosVersion != "" {
			m.eosVersion = msg.eosVersion
		}
		// Apply per-component results independently so a failure in one
		// component does not hide data from the others.
		m.nodeStatsErr = msg.statsErr
		if msg.statsErr == nil {
			m.nodeStats = msg.stats
		}
		m.fstsErr = msg.fstsErr
		if msg.fstsErr == nil {
			m.fsts = msg.fsts
			m.fstSelected = clampIndex(m.fstSelected, len(m.visibleFSTs()))
		}
		m.mgmsErr = msg.mgmsErr
		if msg.mgmsErr == nil {
			m.mgms = msg.mgms
		}
		m.fileSystemsErr = msg.fsErr
		if msg.fsErr == nil {
			m.fileSystems = msg.fs
			m.fsSelected = clampIndex(m.fsSelected, len(m.visibleFileSystems()))
		}
		// Legacy single-error path (early-return failures).
		if msg.err != nil {
			if m.nodeStatsErr == nil {
				m.nodeStatsErr = msg.err
			}
			if m.fstsErr == nil {
				m.fstsErr = msg.err
			}
			if m.mgmsErr == nil {
				m.mgmsErr = msg.err
			}
			if m.fileSystemsErr == nil {
				m.fileSystemsErr = msg.err
			}
			m.status = fmt.Sprintf("Infrastructure refresh failed: %v", msg.err)
		} else {
			m.status = fmt.Sprintf("Connected to %s", m.endpoint)
		}
	case eosVersionLoadedMsg:
		if msg.version != "" {
			m.eosVersion = msg.version
		}
	case nodeStatsLoadedMsg:
		m.fstStatsLoading = false
		m.nodeStatsErr = msg.err
		if msg.err != nil {
			m.status = fmt.Sprintf("Cluster summary refresh failed: %v", msg.err)
		} else {
			m.nodeStats = msg.stats
			m.status = fmt.Sprintf("Connected to %s", m.endpoint)
		}
	case fstsLoadedMsg:
		m.fstsLoading = false
		m.fstsErr = msg.err
		if msg.err != nil {
			m.status = fmt.Sprintf("Node list refresh failed: %v", msg.err)
		} else {
			m.fsts = msg.fsts
			m.fstSelected = clampIndex(m.fstSelected, len(m.visibleFSTs()))
			m.status = fmt.Sprintf("Connected to %s", m.endpoint)
		}
		m.nodeStats.State = m.computeClusterHealth()
	case fileSystemsLoadedMsg:
		m.fileSystemsLoading = false
		m.fileSystemsErr = msg.err
		if msg.err != nil {
			m.status = fmt.Sprintf("Filesystem refresh failed: %v", msg.err)
		} else {
			m.fileSystems = msg.fs
			m.fsSelected = clampIndex(m.fsSelected, len(m.visibleFileSystems()))
			m.status = fmt.Sprintf("Connected to %s", m.endpoint)
		}
		m.nodeStats.State = m.computeClusterHealth()
	case spacesLoadedMsg:
		m.spacesLoading = false
		m.spacesErr = msg.err
		if msg.err != nil {
			m.status = fmt.Sprintf("Spaces refresh failed: %v", msg.err)
		} else {
			m.spaces = msg.spaces
			m.spacesSelected = clampIndex(m.spacesSelected, len(m.spaces))
		}
	case groupsLoadedMsg:
		m.groupsLoading = false
		m.groupsErr = msg.err
		if msg.err != nil {
			m.status = fmt.Sprintf("Groups refresh failed: %v", msg.err)
		} else {
			m.groups = msg.groups
			m.groupsSelected = clampIndex(m.groupsSelected, len(m.visibleGroups()))
			m.status = fmt.Sprintf("Connected to %s", m.endpoint)
		}
	case namespaceStatsLoadedMsg:
		m.nsStatsLoading = false
		m.nsStatsErr = msg.err
		if msg.err != nil {
			m.status = fmt.Sprintf("Namespace stats refresh failed: %v", msg.err)
		} else {
			m.namespaceStats = msg.stats
			m.status = fmt.Sprintf("Connected to %s", m.endpoint)
		}
	case directoryLoadedMsg:
		m.nsLoading = false
		m.nsErr = msg.err
		if msg.err != nil {
			m.status = fmt.Sprintf("Namespace refresh failed: %v", msg.err)
		} else {
			m.nsLoaded = true
			m.directory = msg.directory
			if m.nsSelected >= len(m.directory.Entries) {
				m.nsSelected = max(0, len(m.directory.Entries)-1)
			}
			m.status = fmt.Sprintf("Browsing namespace %s", m.directory.Path)
		}
	case spaceStatusLoadedMsg:
		m.spaceStatusLoading = false
		m.spaceStatusErr = msg.err
		if msg.err != nil {
			m.status = fmt.Sprintf("Space status refresh failed: %v", msg.err)
		} else {
			m.spaceStatus = msg.records
			m.spaceStatusSelected = clampIndex(m.spaceStatusSelected, len(m.spaceStatus))
			m.status = fmt.Sprintf("Connected to %s", m.endpoint)
		}
	case spaceConfigResultMsg:
		m.edit.active = false
		if msg.err != nil {
			m.status = m.styles.error.Render(fmt.Sprintf("Space config failed: %v", msg.err))
		} else {
			m.status = "Space configuration updated successfully"
			return m, loadSpaceStatusCmd(m.client)
		}
	case fsConfigStatusResultMsg:
		m.fsEdit.active = false
		if msg.err != nil {
			m.alert = errorAlert{
				active:  true,
				message: fmt.Sprintf("fs config failed: %v", msg.err),
			}
		} else {
			m.status = fmt.Sprintf("Filesystem %d configstatus updated", m.fsEdit.fsID)
			return m, loadFileSystemsCmd(m.client)
		}
	case ioShapingLoadedMsg:
		m.ioShapingLoading = false
		if msg.err != nil {
			m.ioShapingErr = msg.err
		} else if msg.mode == m.ioShapingMode {
			m.ioShaping = msg.records
			m.ioShapingErr = nil
			m.ioShapingSelected = clampIndex(m.ioShapingSelected, len(m.ioShapingMergedRows()))
		}
	case ioShapingPoliciesLoadedMsg:
		if msg.err == nil {
			m.ioShapingPolicies = msg.records
			m.ioShapingSelected = clampIndex(m.ioShapingSelected, len(m.ioShapingMergedRows()))
		}
	case ioShapingTickMsg:
		if m.activeView == viewIOShaping && !m.ioShapingLoading {
			m.ioShapingLoading = true
			return m, tea.Batch(loadIOShapingCmd(m.client, m.ioShapingMode), ioShapingTickCmd())
		} else if m.activeView == viewIOShaping {
			return m, ioShapingTickCmd()
		}
	case ioShapingPolicyTickMsg:
		if m.activeView == viewIOShaping {
			return m, tea.Batch(loadIOShapingPoliciesCmd(m.client), ioShapingPolicyTickCmd())
		}
	case logLoadedMsg:
		m.log.loading = false
		m.log.err = msg.err
		if msg.err == nil {
			m.log.allLines = msg.lines
			m.log.filtered = applyLogFilter(msg.lines, m.log.filter)
			m.log.vp.SetContent(strings.Join(m.log.filtered, "\n"))
			m.log.vp.GotoBottom()
		}
	case tickMsg:
		return m, tea.Batch(tickCmd(), loadInfraCmd(m.client))
	}

	return m, nil
}

func (m model) View() string {
	header := m.renderHeader()
	footer := m.renderFooter()
	bodyHeight := max(4, m.height-lipgloss.Height(header)-lipgloss.Height(footer)-2)

	if m.log.active {
		body := m.renderLogOverlay(bodyHeight)
		return m.styles.app.Render(header + "\n" + body + "\n" + footer)
	}

	body := m.renderBody(bodyHeight)
	if m.popup.active {
		body = m.renderBodyWithPopup(body, bodyHeight)
	} else if m.edit.active {
		body = m.renderBodyWithEditPopup(body, bodyHeight)
	} else if m.fsEdit.active {
		body = m.renderOverlay(body, m.renderFSConfigStatusEditPopup(), bodyHeight)
	} else if m.alert.active {
		body = m.renderOverlay(body, m.renderErrorAlert(), bodyHeight)
	}

	return m.styles.app.Render(header + "\n" + body + "\n" + footer)
}

func (m model) renderBodyWithPopup(body string, height int) string {
	return m.renderOverlay(body, m.renderFilterPopup(), height)
}

func (m model) renderBodyWithEditPopup(body string, height int) string {
	var popup string
	if m.edit.stage == editStageInput {
		popup = m.renderSpaceStatusEditPopup()
	} else if m.edit.stage == editStageConfirm {
		popup = m.renderSpaceStatusConfirmPopup()
	}
	return m.renderOverlay(body, popup, height)
}

func (m model) onViewChanged() (tea.Model, tea.Cmd) {
	switch m.activeView {
	case viewNamespace:
		return m.maybeLoadNamespace()
	case viewSpaces:
		if !m.spacesLoading && len(m.spaces) == 0 && m.spacesErr == nil {
			m.spacesLoading = true
			m.spacesErr = nil
			return m, loadSpacesCmd(m.client)
		}
		return m, nil
	case viewGroups:
		if !m.groupsLoading && len(m.groups) == 0 && m.groupsErr == nil {
			m.groupsLoading = true
			m.groupsErr = nil
			return m, loadGroupsCmd(m.client)
		}
		return m, nil
	case viewNamespaceStats:
		if !m.nsStatsLoading && m.namespaceStats == (eos.NamespaceStats{}) && m.nsStatsErr == nil {
			m.nsStatsLoading = true
			m.nsStatsErr = nil
			return m, loadNamespaceStatsCmd(m.client)
		}
		return m, nil
	case viewSpaceStatus:
		return m.maybeLoadSpaceStatus()
	case viewIOShaping:
		m.ioShapingLoading = true
		m.ioShapingErr = nil
		return m, tea.Batch(loadIOShapingCmd(m.client, m.ioShapingMode), ioShapingTickCmd(), loadIOShapingPoliciesCmd(m.client), ioShapingPolicyTickCmd())
	default:
		return m, nil
	}
}

func (m model) refreshActiveView() (tea.Model, tea.Cmd) {
	switch m.activeView {
	case viewNamespace:
		m.nsLoaded = false
		m.nsLoading = true
		m.nsErr = nil
		m.status = fmt.Sprintf("Refreshing namespace %s...", m.directory.Path)
		return m, loadDirectoryCmd(m.client, m.directory.Path)
	case viewSpaces:
		m.spacesLoading = true
		m.spacesErr = nil
		m.status = "Refreshing spaces..."
		return m, loadSpacesCmd(m.client)
	case viewGroups:
		m.groupsLoading = true
		m.groupsErr = nil
		m.status = "Refreshing groups..."
		return m, loadGroupsCmd(m.client)
	case viewNamespaceStats:
		m.nsStatsLoading = true
		m.nsStatsErr = nil
		m.status = "Refreshing namespace stats..."
		return m, loadNamespaceStatsCmd(m.client)
	case viewSpaceStatus:
		m.spaceStatusLoading = true
		m.spaceStatusErr = nil
		m.status = "Refreshing space status..."
		return m, loadSpaceStatusCmd(m.client)
	case viewIOShaping:
		m.ioShapingLoading = true
		m.ioShapingErr = nil
		m.status = "Refreshing IO shaping..."
		return m, tea.Batch(loadIOShapingCmd(m.client, m.ioShapingMode), loadIOShapingPoliciesCmd(m.client))
	default:
		m.fstStatsLoading = true
		m.fstsLoading = true
		m.mgmsLoading = true
		m.fileSystemsLoading = true
		m.spacesLoading = true
		m.nsStatsLoading = true
		m.nodeStatsErr = nil
		m.fstsErr = nil
		m.mgmsErr = nil
		m.fileSystemsErr = nil
		m.spacesErr = nil
		m.nsStatsErr = nil
		m.status = "Refreshing..."
		return m, loadInfraCmd(m.client)
	}
}

func (m model) maybeLoadNamespace() (tea.Model, tea.Cmd) {
	if m.nsLoaded || m.nsLoading {
		return m, nil
	}

	m.nsLoading = true
	m.nsErr = nil
	m.status = fmt.Sprintf("Loading namespace %s...", m.directory.Path)
	return m, loadDirectoryCmd(m.client, m.directory.Path)
}

func (m model) maybeLoadSpaceStatus() (tea.Model, tea.Cmd) {
	if !m.spaceStatusLoading && len(m.spaceStatus) > 0 {
		return m, nil
	}

	m.spaceStatusLoading = true
	m.spaceStatusErr = nil
	m.status = "Loading space status..."
	return m, loadSpaceStatusCmd(m.client)
}

func (m model) computeClusterHealth() string {
	fsts := m.fsts
	fss := m.fileSystems
	if len(fsts) == 0 && len(fss) == 0 {
		return "-"
	}
	for _, node := range fsts {
		if strings.ToLower(node.Status) != "online" {
			return "WARN"
		}
	}
	for _, fs := range fss {
		if strings.ToLower(fs.Boot) != "booted" {
			return "WARN"
		}
	}
	return "OK"
}
