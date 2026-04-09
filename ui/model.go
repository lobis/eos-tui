package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/lobis/eos-tui/eos"
)

const refreshInterval = 5 * time.Second

type viewID int

const (
	viewMGM            viewID = iota // tab 1: MGM nodes (EOS version)
	viewQDB                          // tab 2: QDB cluster (raft-info)
	viewFST                          // tab 3: FST nodes
	viewFileSystems                  // tab 4: File systems
	viewNamespace                    // tab 5: Namespace browser
	viewSpaces                       // tab 6: Spaces
	viewNamespaceStats               // tab 7: NS stats
	viewSpaceStatus                  // tab 8: Space status
	viewIOShaping                    // tab 9: IO Traffic
	viewGroups                       // tab 0: Groups
)

const viewCount = 10

type infraLoadedMsg struct {
	stats      eos.NodeStats
	fsts       []eos.FstRecord
	mgms       []eos.MgmRecord
	fs         []eos.FileSystemRecord
	eosVersion string
	// Per-component errors so a failure in one component does not hide others.
	statsErr error
	fstsErr  error
	mgmsErr  error
	fsErr    error
	// Legacy single-error field kept for callers that return early.
	err error
}

type nodeStatsLoadedMsg struct {
	stats eos.NodeStats
	err   error
}

type fstsLoadedMsg struct {
	fsts []eos.FstRecord
	err  error
}

type fileSystemsLoadedMsg struct {
	fs  []eos.FileSystemRecord
	err error
}

type mgmsLoadedMsg struct {
	mgms []eos.MgmRecord
	err  error
}

type spacesLoadedMsg struct {
	spaces []eos.SpaceRecord
	err    error
}

type groupsLoadedMsg struct {
	groups []eos.GroupRecord
	err    error
}

type namespaceStatsLoadedMsg struct {
	stats eos.NamespaceStats
	err   error
}

type directoryLoadedMsg struct {
	directory eos.Directory
	err       error
}

type spaceStatusLoadedMsg struct {
	records []eos.SpaceStatusRecord
	err     error
}

type spaceConfigResultMsg struct {
	err error
}

type ioShapingLoadedMsg struct {
	records []eos.IOShapingRecord
	mode    eos.IOShapingMode
	err     error
}

type ioShapingPoliciesLoadedMsg struct {
	records []eos.IOShapingPolicyRecord
	err     error
}

type eosVersionLoadedMsg struct {
	version string
}

type logLoadedMsg struct {
	filePath string
	lines    []string
	err      error
}

type ioShapingTickMsg struct{}
type ioShapingPolicyTickMsg struct{}

type tickMsg time.Time

type model struct {
	client   *eos.Client
	endpoint string

	width  int
	height int

	activeView viewID

	fstStatsLoading    bool
	fstsLoading        bool
	fileSystemsLoading bool
	nodeStatsErr       error
	fstsErr            error
	fileSystemsErr     error
	nodeStats          eos.NodeStats
	fsts               []eos.FstRecord

	mgmsLoading bool
	mgmsErr     error
	mgms        []eos.MgmRecord

	eosVersion string

	mgmSelected int
	qdbSelected int

	fstSelected       int
	fstColumnSelected int
	fileSystems       []eos.FileSystemRecord
	fsSelected        int
	fsColumnSelected  int

	spaces               []eos.SpaceRecord
	spacesLoading        bool
	spacesErr            error
	spacesSelected       int
	spacesColumnSelected int

	groups               []eos.GroupRecord
	groupsLoading        bool
	groupsErr            error
	groupsSelected       int
	groupsColumnSelected int

	namespaceStats eos.NamespaceStats
	nsStatsLoading bool
	nsStatsErr     error

	directory  eos.Directory
	nsLoaded   bool
	nsLoading  bool
	nsErr      error
	nsSelected int

	spaceStatus         []eos.SpaceStatusRecord
	spaceStatusLoading  bool
	spaceStatusErr      error
	spaceStatusSelected int

	ioShaping         []eos.IOShapingRecord
	ioShapingPolicies []eos.IOShapingPolicyRecord
	ioShapingMode     eos.IOShapingMode
	ioShapingLoading  bool
	ioShapingErr      error
	ioShapingSelected int

	status string

	fstFilter   filterState
	fstSort     sortState
	fsFilter    filterState
	fsSort      sortState
	groupFilter filterState
	groupSort   sortState
	popup       filterPopup
	edit        spaceStatusEdit
	fsEdit      fsConfigStatusEdit
	alert       errorAlert
	log         logOverlay

	styles styles
}

type fsConfigStatusResultMsg struct {
	err error
}

type errorAlert struct {
	active  bool
	message string
}

type fsConfigStatusEdit struct {
	active   bool
	fsID     uint64
	fsPath   string
	current  string
	selected int // index into configStatusOptions
}

var configStatusOptions = []string{"rw", "ro", "drain", "empty"}

type spaceStatusEditStage int

const (
	editStageNone spaceStatusEditStage = iota
	editStageInput
	editStageConfirm
)

type buttonID int

const (
	buttonCancel buttonID = iota
	buttonContinue
)

type spaceStatusEdit struct {
	active     bool
	stage      spaceStatusEditStage
	record     eos.SpaceStatusRecord
	input      textinput.Model
	button     buttonID
	focusInput bool
}

type filterPopup struct {
	active bool
	view   viewID
	column int
	input  textinput.Model
	table  table.Model
	values []string
}

type logOverlay struct {
	active    bool
	host      string // specific host to read from (empty = effective target)
	filePath  string
	title     string
	allLines  []string // raw lines from tail
	filtered  []string // lines matching current filter
	filter    string   // current grep string
	filtering bool     // filter input is active
	vp        viewport.Model
	input     textinput.Model
	err       error
	loading   bool
}

type filterState struct {
	column  int
	filters map[int]string
}

type sortState struct {
	column int
	desc   bool
}

type fstFilterColumn int
type fstSortColumn int
type mgmFilterColumn int
type qdbFilterColumn int
type fsFilterColumn int
type fsSortColumn int
type groupFilterColumn int
type groupSortColumn int

const (
	fstFilterHost           fstFilterColumn = iota // 0 — visible column 0
	fstFilterPort                                  // 1 — visible column 1
	fstFilterGeotag                                // 2 — visible column 2
	fstFilterStatus                                // 3 — visible column 3
	fstFilterActivated                             // 4 — visible column 4
	fstFilterHeartbeatDelta                        // 5 — visible column 5
	fstFilterNoFS                                  // 6 — visible column 6
	fstFilterEOSVersion                            // 7 — visible column 7
	fstFilterType                                  // 8 — not navigable (hidden field)
)

const fstSortNone fstSortColumn = -1

const (
	fstSortHost       fstSortColumn = iota // 0 — visible column 0
	fstSortPort                            // 1
	fstSortGeotag                          // 2
	fstSortStatus                          // 3
	fstSortActivated                       // 4
	fstSortHeartbeat                       // 5
	fstSortNoFS                            // 6
	fstSortEOSVersion                      // 7
	fstSortType                            // 8 — not navigable
)

const fstSortFileSystems = fstSortNoFS

const (
	fsFilterHost fsFilterColumn = iota
	fsFilterPort
	fsFilterID
	fsFilterPath
	fsFilterGroup
	fsFilterGeotag
	fsFilterBoot
	fsFilterConfigStatus
	fsFilterDrain
	fsFilterUsage
	fsFilterStatus
	fsFilterHealth
)

const (
	mgmFilterHost mgmFilterColumn = iota
	mgmFilterPort
	mgmFilterRole
	mgmFilterStatus
	mgmFilterEOSVersion
)

const (
	qdbFilterHost qdbFilterColumn = iota
	qdbFilterPort
	qdbFilterRole
	qdbFilterStatus
	qdbFilterVersion
)

const groupSortNone groupSortColumn = -1

const (
	groupFilterName groupFilterColumn = iota
	groupFilterStatus
	groupFilterNoFS
	groupFilterCapacity
	groupFilterUsed
	groupFilterFree
	groupFilterFiles
)

const (
	groupSortName groupSortColumn = iota
	groupSortStatus
	groupSortNoFS
	groupSortCapacity
	groupSortUsed
	groupSortFree
	groupSortFiles
)

const fsSortNone fsSortColumn = -1

const (
	fsSortHost fsSortColumn = iota
	fsSortPort
	fsSortID
	fsSortPath
	fsSortGroup
	fsSortGeotag
	fsSortBoot
	fsSortConfigStatus
	fsSortDrain
	fsSortUsed
	fsSortStatus
	fsSortHealth
)

type styles struct {
	app       lipgloss.Style
	header    lipgloss.Style
	tab       lipgloss.Style
	tabActive lipgloss.Style
	panel     lipgloss.Style
	panelDim  lipgloss.Style
	selected  lipgloss.Style
	label     lipgloss.Style
	value     lipgloss.Style
	error     lipgloss.Style
	status    lipgloss.Style
}

type tableColumn struct {
	title  string
	min    int
	maxw   int // 0 = no max
	weight int
	right  bool
}

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

func (m model) renderOverlay(body string, popup string, height int) string {
	bodyLines := strings.Split(body, "\n")
	popupLines := strings.Split(popup, "\n")
	width := m.contentWidth()

	for len(bodyLines) < height {
		bodyLines = append(bodyLines, strings.Repeat(" ", width))
	}

	popupHeight := len(popupLines)
	popupWidth := 0
	for _, line := range popupLines {
		popupWidth = max(popupWidth, lipgloss.Width(line))
	}
	popupWidth = min(popupWidth, width)
	topPad := max(0, (height-popupHeight)/2)
	leftPad := max(0, (width-popupWidth)/2)

	for i := 0; i < popupHeight && topPad+i < len(bodyLines); i++ {
		bodyLine := padVisibleWidth(bodyLines[topPad+i], width)
		popupLine := padVisibleWidth(popupLines[i], popupWidth)
		left := ansi.Cut(bodyLine, 0, leftPad)
		right := ansi.Cut(bodyLine, leftPad+popupWidth, width)
		bodyLines[topPad+i] = left + popupLine + right
	}

	if len(bodyLines) > height {
		bodyLines = bodyLines[:height]
	}
	return strings.Join(bodyLines, "\n")
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
	switch msg.String() {
	case "up", "k":
		if m.nsSelected > 0 {
			m.nsSelected--
		}
	case "down", "j":
		if m.nsSelected < len(m.directory.Entries)-1 {
			m.nsSelected++
		}
	case "ctrl+u":
		m.nsSelected = max(0, m.nsSelected-half)
		return m, nil
	case "ctrl+d":
		m.nsSelected = min(len(m.directory.Entries)-1, m.nsSelected+half)
		return m, nil
	case "G":
		m.nsSelected = max(0, len(m.directory.Entries)-1)
		return m, nil
	case "g":
		m.nsSelected = 0
		m.nsLoading = true
		m.status = "Jumping to / ..."
		return m, loadDirectoryCmd(m.client, "/")
	case "backspace", "left":
		parent := parentPath(m.directory.Path)
		if parent != m.directory.Path {
			m.nsSelected = 0
			m.nsLoading = true
			m.status = fmt.Sprintf("Opening %s...", parent)
			return m, loadDirectoryCmd(m.client, parent)
		}
	case "enter", "right":
		entry, ok := m.selectedNamespaceEntry()
		if ok && entry.Kind == eos.EntryKindContainer {
			m.nsSelected = 0
			m.nsLoading = true
			m.status = fmt.Sprintf("Opening %s...", entry.Path)
			return m, loadDirectoryCmd(m.client, entry.Path)
		}
	}

	return m, nil
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
	}
	m.ioShapingSelected = clampIndex(m.ioShapingSelected, n)
	return m, nil
}

func (m model) selectedNode() (eos.FstRecord, bool) {
	fsts := m.visibleFSTs()
	if len(fsts) == 0 || m.fstSelected < 0 || m.fstSelected >= len(fsts) {
		return eos.FstRecord{}, false
	}

	return fsts[m.fstSelected], true
}

func (m model) selectedFileSystem() (eos.FileSystemRecord, bool) {
	fileSystems := m.visibleFileSystems()
	if len(fileSystems) == 0 || m.fsSelected < 0 || m.fsSelected >= len(fileSystems) {
		return eos.FileSystemRecord{}, false
	}

	return fileSystems[m.fsSelected], true
}

func (m model) selectedNamespaceEntry() (eos.Entry, bool) {
	if len(m.directory.Entries) == 0 || m.nsSelected < 0 || m.nsSelected >= len(m.directory.Entries) {
		return eos.Entry{}, false
	}

	return m.directory.Entries[m.nsSelected], true
}

func (m model) selectedSpaceStatusRecord() (eos.SpaceStatusRecord, bool) {
	if len(m.spaceStatus) == 0 || m.spaceStatusSelected < 0 || m.spaceStatusSelected >= len(m.spaceStatus) {
		return eos.SpaceStatusRecord{}, false
	}
	return m.spaceStatus[m.spaceStatusSelected], true
}

func (m model) startSpaceStatusEdit() (tea.Model, tea.Cmd) {
	record, ok := m.selectedSpaceStatusRecord()
	if !ok {
		return m, nil
	}

	input := textinput.New()
	input.Placeholder = "new value"
	input.Focus()
	input.SetValue(record.Value)

	m.edit = spaceStatusEdit{
		active:     true,
		stage:      editStageInput,
		record:     record,
		input:      input,
		button:     buttonCancel,
		focusInput: true,
	}

	return m, nil
}

func (m model) updateSpaceStatusEditKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.edit.active = false
		return m, nil
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

func (m model) openFSConfigStatusEdit() (tea.Model, tea.Cmd) {
	fs, ok := m.selectedFileSystem()
	if !ok {
		return m, nil
	}
	// Find starting index matching the current configstatus.
	sel := 0
	for i, opt := range configStatusOptions {
		if fs.ConfigStatus == opt {
			sel = i
			break
		}
	}
	m.fsEdit = fsConfigStatusEdit{
		active:   true,
		fsID:     fs.ID,
		fsPath:   fs.Path,
		current:  fs.ConfigStatus,
		selected: sel,
	}
	return m, nil
}

func (m model) updateFSConfigStatusEditKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.fsEdit.active = false
		return m, nil
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

func (m model) renderFSConfigStatusEditPopup() string {
	lines := []string{
		m.styles.label.Render("Set configstatus"),
		fmt.Sprintf("Filesystem: %s (id %d)", m.fsEdit.fsPath, m.fsEdit.fsID),
		fmt.Sprintf("Current:    %s", m.styles.value.Render(fallback(m.fsEdit.current, "-"))),
		"",
	}
	for i, opt := range configStatusOptions {
		if i == m.fsEdit.selected {
			lines = append(lines, m.styles.selected.Render("▶ "+opt))
		} else {
			lines = append(lines, "  "+opt)
		}
	}
	lines = append(lines, "", m.styles.status.Render("↑↓ select  •  enter apply  •  esc cancel"))
	return m.styles.panel.
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m model) renderErrorAlert() string {
	lines := []string{
		m.styles.error.Render("Error"),
		"",
		m.alert.message,
		"",
		m.styles.status.Render("enter / esc  close"),
	}
	return m.styles.panel.
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func runFsConfigStatusCmd(client *eos.Client, fsID uint64, value string) tea.Cmd {
	return func() tea.Msg {
		err := client.FsConfigStatus(context.Background(), fsID, value)
		return fsConfigStatusResultMsg{err: err}
	}
}

func (m model) renderHeader() string {
	type tabDef struct {
		label string
		view  viewID
	}
	tabs := []tabDef{
		{"1 MGM", viewMGM},
		{"2 QDB", viewQDB},
		{"3 FST", viewFST},
		{"4 FS", viewFileSystems},
		{"5 Namespace", viewNamespace},
		{"6 Spaces", viewSpaces},
		{"7 NS Stats", viewNamespaceStats},
		{"8 Space Status", viewSpaceStatus},
		{"9 IO Traffic", viewIOShaping},
		{"0 Groups", viewGroups},
	}

	parts := []string{m.styles.header.Render("EOS TUI"), "  "}
	for i, t := range tabs {
		if i > 0 {
			parts = append(parts, " ")
		}
		if m.activeView == t.view {
			parts = append(parts, m.styles.tabActive.Render(t.label))
		} else {
			parts = append(parts, m.styles.tab.Render(t.label))
		}
	}

	left := lipgloss.JoinHorizontal(lipgloss.Left, parts...)
	right := m.styles.label.Render("target ") + m.styles.value.Render(m.endpoint)
	spacerWidth := max(1, m.contentWidth()-lipgloss.Width(left)-lipgloss.Width(right))

	return lipgloss.JoinHorizontal(lipgloss.Left, left, strings.Repeat(" ", spacerWidth), right)
}

// renderMGMView shows the MGM nodes with their EOS server version.
// The MGM hosts are the same nodes that participate in the QDB cluster.
// The EOS server version is fetched via `eos version` (applies to all MGMs).
func (m model) renderMGMView(height int) string {
	width := m.contentWidth()
	contentWidth := panelContentWidth(width)

	mgms := m.mgms

	columns := allocateTableColumns(contentWidth, []tableColumn{
		{title: "host", min: 15, weight: 1},
		{title: "port", min: 5, weight: 0, right: true},
		{title: "role", min: 10, weight: 0},
		{title: "status", min: 10, weight: 0},
		{title: "eos version", min: 14, weight: 0},
	})

	title := m.styles.label.Render("management nodes (mgm)")
	lines := []string{
		title,
		"",
		m.renderSimpleHeaderRow(columns, []string{"host", "port", "role", "status", "eos version"}),
	}

	if m.mgmsLoading && len(mgms) == 0 {
		lines = append(lines, "loading mgm info...")
	} else if len(mgms) == 0 {
		lines = append(lines, "(no mgm nodes found)")
	} else {
		for i, node := range mgms {
			row := formatTableRow(columns, []string{
				node.Host,
				fmt.Sprintf("%d", node.Port),
				strings.ToLower(node.Role),
				strings.ToLower(node.Status),
				m.eosVersion,
			})
			if i == m.mgmSelected {
				row = m.styles.selected.Width(contentWidth).Render(row)
			}
			lines = append(lines, row)
		}
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, height))
}

// renderQDBView shows the QDB cluster topology from `redis-cli raft-info`.
func (m model) renderQDBView(height int) string {
	width := m.contentWidth()
	contentWidth := panelContentWidth(width)

	mgms := m.mgms

	columns := allocateTableColumns(contentWidth, []tableColumn{
		{title: "host", min: 15, weight: 1},
		{title: "port", min: 5, weight: 0, right: true},
		{title: "role", min: 10, weight: 0},
		{title: "status", min: 10, weight: 0},
		{title: "qdb version", min: 14, weight: 0},
	})

	title := m.styles.label.Render("quarkdb cluster (qdb)")
	lines := []string{
		title,
		"",
		m.renderSimpleHeaderRow(columns, []string{"host", "port", "role", "status", "qdb version"}),
	}

	if m.mgmsLoading && len(mgms) == 0 {
		lines = append(lines, "loading qdb info...")
	} else if m.mgmsErr != nil {
		lines = append(lines, m.styles.error.Render(m.mgmsErr.Error()))
	} else if len(mgms) == 0 {
		lines = append(lines, "(no qdb nodes found)")
	} else {
		for i, node := range mgms {
			row := formatTableRow(columns, []string{
				node.QDBHost,
				fmt.Sprintf("%d", node.QDBPort),
				strings.ToLower(node.Role),
				strings.ToLower(node.Status),
				node.EOSVersion,
			})
			if i == m.qdbSelected {
				row = m.styles.selected.Width(contentWidth).Render(row)
			}
			lines = append(lines, row)
		}
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, height))
}

func (m model) renderBody(availableHeight int) string {
	switch m.activeView {
	case viewMGM:
		return m.renderMGMView(availableHeight)
	case viewQDB:
		return m.renderQDBView(availableHeight)
	case viewFST:
		return m.renderFSTView(availableHeight)
	case viewFileSystems:
		return m.renderFileSystemsView(availableHeight)
	case viewNamespace:
		return m.renderNamespaceView(availableHeight)
	case viewSpaces:
		return m.renderSpacesView(availableHeight)
	case viewNamespaceStats:
		return m.renderNamespaceStatsView(availableHeight)
	case viewSpaceStatus:
		return m.renderSpaceStatusView(availableHeight)
	case viewIOShaping:
		return m.renderIOShapingView(availableHeight)
	case viewGroups:
		return m.renderGroupsView(availableHeight)
	default:
		return ""
	}
}

func (m model) renderFSTView(height int) string {
	filterLines := 0
	if len(m.fstFilter.filters) > 0 {
		filterLines = 1
	}
	fixedHeaderLines := 6 + filterLines // title+controls, 3 metric lines, blank, col headers [, filters]
	naturalListContent := fixedHeaderLines + len(m.visibleFSTs())
	const fstDetailLines = 18 // fixed lines rendered by renderNodeDetails
	listHeight, detailHeight := adaptiveSplitHeights(height, naturalListContent, fstDetailLines)
	width := m.contentWidth()

	list := m.renderNodesList(width, listHeight)
	details := m.renderNodeDetails(width, detailHeight)
	return lipgloss.JoinVertical(lipgloss.Left, list, details)
}

func (m model) renderNodesList(width, height int) string {
	contentWidth := panelContentWidth(width)
	fsts := m.visibleFSTs()

	// Build data rows first so column widths can be fitted to content.
	dataRows := make([][]string, len(fsts))
	for i, node := range fsts {
		dataRows[i] = []string{
			node.Host,
			fmt.Sprintf("%d", node.Port),
			node.Geotag,
			node.Status,
			node.Activated,
			fmt.Sprintf("%d", node.HeartbeatDelta),
			fmt.Sprintf("%d", node.FileSystemCount),
			node.EOSVersion,
		}
	}
	columnDefs := contentAwareColumns([]tableColumn{
		{title: "host", min: 8, weight: 5},
		{title: "port", min: 5, weight: 0, right: true},
		{title: "geotag", min: 6, weight: 3},
		{title: "status", min: 6, weight: 0},
		{title: "activated", min: 9, weight: 0},
		{title: "heartbeatdelta", min: 14, weight: 0, right: true},
		{title: "nofs", min: 4, weight: 0, right: true},
		{title: "eos version", min: 11, weight: 0},
	}, dataRows)
	columns := allocateTableColumns(contentWidth, columnDefs)

	title := m.styles.label.Render("Cluster Summary")
	lines := []string{
		title + m.renderNodeControls(),
		m.metricLine("Health", fallback(m.nodeStats.State, "-"), "Threads", fmt.Sprintf("%d", m.nodeStats.ThreadCount)),
		m.metricLine("Files", fmt.Sprintf("%d", m.nodeStats.FileCount), "Dirs", fmt.Sprintf("%d", m.nodeStats.DirCount)),
		m.metricLine("Uptime", formatDuration(m.nodeStats.Uptime), "FDs", fmt.Sprintf("%d", m.nodeStats.FileDescs)),
		"",
		m.renderFstHeaderRow(columns),
	}

	if m.fstStatsLoading {
		lines[1] = m.styles.value.Render("Loading cluster summary...")
		lines[2] = ""
		lines[3] = ""
	}
	if summary := m.renderFilterSummary(m.fstFilter.filters, func(col int) string {
		old := m.fstFilter.column
		m.fstFilter.column = col
		label := m.fstFilterColumnLabel()
		m.fstFilter.column = old
		return label
	}); summary != "" {
		lines = append(lines, summary)
	}

	if m.fstsLoading {
		lines = append(lines, "Loading node list...")
	} else if m.fstsErr != nil {
		lines = append(lines, m.styles.error.Render(m.fstsErr.Error()))
	} else if len(fsts) == 0 {
		lines = append(lines, "(no fsts)")
	} else {
		start, end := visibleWindow(len(fsts), m.fstSelected, max(1, panelContentHeight(height)-len(lines)))
		lines[0] = title + m.renderNodeControls() + renderScrollSummary(start, end, len(fsts))
		for i := start; i < end; i++ {
			line := formatTableRow(columns, dataRows[i])
			if i == m.fstSelected {
				line = m.styles.selected.Width(contentWidth).Render(line)
			}
			lines = append(lines, line)
		}
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, panelContentHeight(height)))
}

func (m model) renderNodeDetails(width, height int) string {
	node, ok := m.selectedNode()
	if !ok {
		return m.styles.panelDim.Width(width).Render(fitLines([]string{"No node selected"}, panelContentHeight(height)))
	}

	lines := []string{
		m.styles.label.Render("Selected Node"),
		truncate(node.Host+":"+fmt.Sprintf("%d", node.Port), max(10, width-4)),
		"",
		m.metricLine("Type", fallback(node.Type, "-"), "EOS", fallback(node.EOSVersion, "-")),
		m.metricLine("Status", fallback(node.Status, "-"), "Activated", fallback(node.Activated, "-")),
		m.metricLine("Geotag", fallback(node.Geotag, "-"), "Filesystems", fmt.Sprintf("%d", node.FileSystemCount)),
		m.metricLine("Heartbeat", fmt.Sprintf("%ds", node.HeartbeatDelta), "Disk Load", fmt.Sprintf("%.2f", node.DiskLoad)),
		m.metricLine("Capacity", humanBytes(node.CapacityBytes), "Used", humanBytes(node.UsedBytes)),
		m.metricLine("Free", humanBytes(node.FreeBytes), "Files", fmt.Sprintf("%d", node.UsedFiles)),
		m.metricLine("RSS", humanBytes(node.RSSBytes), "VSize", humanBytes(node.VSizeBytes)),
		m.metricLine("Threads", fmt.Sprintf("%d", node.ThreadCount), "Read MB/s", fmt.Sprintf("%.2f", node.ReadRateMB)),
		m.metricLine("Write MB/s", fmt.Sprintf("%.2f", node.WriteRateMB), "", ""),
		"",
		m.styles.label.Render("Uptime"),
		truncate(strings.ReplaceAll(node.Uptime, "%20", " "), max(10, width-4)),
		"",
		m.styles.label.Render("Kernel"),
		truncate(node.Kernel, max(10, width-4)),
	}

	return m.styles.panelDim.Width(width).Render(fitLines(lines, panelContentHeight(height)))
}

func (m model) renderFileSystemsView(height int) string {
	filterLines := 0
	if len(m.fsFilter.filters) > 0 {
		filterLines = 1
	}
	fixedHeaderLines := 3 + filterLines // title+controls, blank, col headers [, filters]
	naturalListContent := fixedHeaderLines + len(m.visibleFileSystems())
	const fsDetailLines = 14 // fixed lines rendered by renderFileSystemDetails
	listHeight, detailHeight := adaptiveSplitHeights(height, naturalListContent, fsDetailLines)
	width := m.contentWidth()

	list := m.renderFileSystemsList(width, listHeight)
	details := m.renderFileSystemDetails(width, detailHeight)
	return lipgloss.JoinVertical(lipgloss.Left, list, details)
}

func (m model) renderFileSystemsList(width, height int) string {
	contentWidth := panelContentWidth(width)
	fileSystems := m.visibleFileSystems()

	// Pre-build data rows so column widths can be fitted to actual content.
	dataRows := make([][]string, len(fileSystems))
	for i, fs := range fileSystems {
		dataRows[i] = []string{
			fs.Host,
			fmt.Sprintf("%d", fs.Port),
			fmt.Sprintf("%d", fs.ID),
			fs.Path,
			fs.SchedGroup,
			fs.Geotag,
			fs.Boot,
			fs.ConfigStatus,
			fs.DrainStatus,
			fmt.Sprintf("%.2f", usagePercent(fs.UsedBytes, fs.CapacityBytes)),
			fs.Active,
			fs.Health,
		}
	}
	columnDefs := contentAwareColumns([]tableColumn{
		{title: "host", min: 4, weight: 4},
		{title: "port", min: 4, weight: 0, right: true},
		{title: "id", min: 2, weight: 0, right: true},
		{title: "path", min: 4, maxw: 28, weight: 3},
		{title: "schedgroup", min: 10, weight: 1},
		{title: "geotag", min: 6, weight: 1},
		{title: "boot", min: 4, weight: 0},
		{title: "configstatus", min: 12, weight: 0},
		{title: "drain", min: 5, weight: 0},
		{title: "usage %", min: 7, weight: 0, right: true},
		{title: "active", min: 6, weight: 0},
		{title: "health", min: 4, weight: 1},
	}, dataRows)
	columns := allocateTableColumns(contentWidth, columnDefs)

	title := m.styles.label.Render("EOS Filesystems")
	lines := []string{
		title + m.renderFileSystemControls(),
		"",
		m.renderFileSystemHeaderRow(columns),
	}
	if summary := m.renderFilterSummary(m.fsFilter.filters, func(col int) string {
		old := m.fsFilter.column
		m.fsFilter.column = col
		label := m.fsFilterColumnLabel()
		m.fsFilter.column = old
		return label
	}); summary != "" {
		lines = append(lines, summary)
	}

	if m.fileSystemsLoading {
		lines = append(lines, "Loading filesystem state...")
	} else if m.fileSystemsErr != nil {
		lines = append(lines, m.styles.error.Render(m.fileSystemsErr.Error()))
	} else if len(fileSystems) == 0 {
		lines = append(lines, "(no filesystems)")
	} else {
		start, end := visibleWindow(len(fileSystems), m.fsSelected, max(1, panelContentHeight(height)-len(lines)))
		lines[0] = title + m.renderFileSystemControls() + renderScrollSummary(start, end, len(fileSystems))
		for i := start; i < end; i++ {
			line := formatTableRow(columns, dataRows[i])
			if i == m.fsSelected {
				line = m.styles.selected.Width(contentWidth).Render(line)
			}
			lines = append(lines, line)
		}
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, panelContentHeight(height)))
}

func (m model) renderFileSystemDetails(width, height int) string {
	fs, ok := m.selectedFileSystem()
	if !ok {
		return m.styles.panelDim.Width(width).Render(fitLines([]string{"No filesystem selected"}, panelContentHeight(height)))
	}

	lines := []string{
		m.styles.label.Render("Selected Filesystem"),
		truncate(fmt.Sprintf("%s:%d", fs.Host, fs.Port), max(10, width-4)),
		"",
		m.metricLine("ID", fmt.Sprintf("%d", fs.ID), "Group", fallback(fs.SchedGroup, "-")),
		m.metricLine("Boot", fallback(fs.Boot, "-"), "Config", fallback(fs.ConfigStatus, "-")),
		m.metricLine("Drain", fallback(fs.DrainStatus, "-"), "Active", fallback(fs.Active, "-")),
		m.metricLine("Geotag", fallback(fs.Geotag, "-"), "Health", truncate(fs.Health, 12)),
		m.metricLine("Capacity", humanBytes(fs.CapacityBytes), "Used", humanBytes(fs.UsedBytes)),
		m.metricLine("Free", humanBytes(fs.FreeBytes), "Files", fmt.Sprintf("%d", fs.UsedFiles)),
		m.metricLine("BW", fmt.Sprintf("%.0f MB/s", fs.DiskBWMB), "IOPS", fmt.Sprintf("%.0f", fs.DiskIOPS)),
		m.metricLine("Read", fmt.Sprintf("%.2f MB/s", fs.ReadRateMB), "Write", fmt.Sprintf("%.2f MB/s", fs.WriteRateMB)),
		"",
		m.styles.label.Render("Mount Path"),
		truncate(fs.Path, max(10, width-4)),
	}

	return m.styles.panelDim.Width(width).Render(fitLines(lines, panelContentHeight(height)))
}

func (m model) renderSpacesView(height int) string {
	const fixedHeaderLines = 3 // title, blank, column headers
	naturalListContent := fixedHeaderLines + len(m.spaces)
	const spaceDetailLines = 8 // fixed lines rendered by renderSpaceDetails
	listHeight, detailHeight := adaptiveSplitHeights(height, naturalListContent, spaceDetailLines)
	width := m.contentWidth()

	list := m.renderSpacesList(width, listHeight)
	details := m.renderSpaceDetails(width, detailHeight)
	return lipgloss.JoinVertical(lipgloss.Left, list, details)
}

func (m model) renderSpacesList(width, height int) string {
	contentWidth := panelContentWidth(width)

	dataRows := make([][]string, len(m.spaces))
	for i, space := range m.spaces {
		dataRows[i] = []string{
			space.Name,
			space.Type,
			space.Status,
			fmt.Sprintf("%d", space.Groups),
			fmt.Sprintf("%d", space.NumFiles),
			fmt.Sprintf("%d", space.NumContainers),
			fmt.Sprintf("%.2f", usagePercent(space.UsedBytes, space.CapacityBytes)),
		}
	}
	columnDefs := contentAwareColumns([]tableColumn{
		{title: "name", min: 4, weight: 3},
		{title: "type", min: 4, weight: 1},
		{title: "status", min: 6, weight: 1},
		{title: "groups", min: 6, weight: 0, right: true},
		{title: "files", min: 5, weight: 0, right: true},
		{title: "dirs", min: 4, weight: 0, right: true},
		{title: "usage %", min: 7, weight: 0, right: true},
	}, dataRows)
	columns := allocateTableColumns(contentWidth, columnDefs)

	title := m.styles.label.Render("EOS Spaces")
	lines := []string{
		title,
		"",
		m.renderSpaceHeaderRow(columns),
	}

	if m.spacesLoading {
		lines = append(lines, "Loading spaces...")
	} else if m.spacesErr != nil {
		lines = append(lines, m.styles.error.Render(m.spacesErr.Error()))
	} else if len(m.spaces) == 0 {
		lines = append(lines, "(no spaces)")
	} else {
		start, end := visibleWindow(len(m.spaces), m.spacesSelected, max(1, panelContentHeight(height)-len(lines)))
		lines[0] = title + renderScrollSummary(start, end, len(m.spaces))
		for i := start; i < end; i++ {
			line := formatTableRow(columns, dataRows[i])
			if i == m.spacesSelected {
				line = m.styles.selected.Width(contentWidth).Render(line)
			}
			lines = append(lines, line)
		}
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, panelContentHeight(height)))
}

func (m model) renderSpaceHeaderRow(columns []tableColumn) string {
	return m.renderSimpleHeaderRow(columns, []string{"name", "type", "status", "groups", "files", "dirs", "usage %"})
}

func (m model) renderSpaceDetails(width, height int) string {
	if len(m.spaces) == 0 || m.spacesSelected >= len(m.spaces) {
		return m.styles.panelDim.Width(width).Render(fitLines([]string{"No space selected"}, panelContentHeight(height)))
	}

	space := m.spaces[m.spacesSelected]

	lines := []string{
		m.styles.label.Render("Selected Space"),
		truncate(space.Name, max(10, width-4)),
		"",
		m.metricLine("Type", space.Type, "Status", space.Status),
		m.metricLine("Groups", fmt.Sprintf("%d", space.Groups), "Files", fmt.Sprintf("%d", space.NumFiles)),
		m.metricLine("Directories", fmt.Sprintf("%d", space.NumContainers), "", ""),
		m.metricLine("Capacity", humanBytes(space.CapacityBytes), "Used", humanBytes(space.UsedBytes)),
		m.metricLine("Free", humanBytes(space.FreeBytes), "", ""),
	}

	return m.styles.panelDim.Width(width).Render(fitLines(lines, panelContentHeight(height)))
}

func (m model) renderNamespaceStatsView(height int) string {
	width := m.contentWidth()
	contentWidth := panelContentWidth(width)

	if m.nsStatsLoading {
		return m.styles.panelDim.Width(width).Render(fitLines([]string{"Loading namespace statistics..."}, height))
	}
	if m.nsStatsErr != nil {
		return m.styles.panelDim.Width(width).Render(fitLines([]string{m.styles.error.Render(m.nsStatsErr.Error())}, height))
	}

	stats := m.namespaceStats
	lines := []string{
		m.styles.label.Render("Namespace Statistics"),
		"",
		m.metricLine("Total Files", fmt.Sprintf("%d", stats.TotalFiles), "Total Directories", fmt.Sprintf("%d", stats.TotalDirectories)),
		"",
		m.styles.label.Render("IDs"),
		m.metricLine("Current File ID", fmt.Sprintf("%d", stats.CurrentFID), "Current Container ID", fmt.Sprintf("%d", stats.CurrentCID)),
		m.metricLine("Generated File IDs", fmt.Sprintf("%d", stats.GeneratedFID), "Generated Container IDs", fmt.Sprintf("%d", stats.GeneratedCID)),
		"",
		m.styles.label.Render("Lock Contention"),
		m.metricLine("Read Contention", fmt.Sprintf("%.2f", stats.ContentionRead), "Write Contention", fmt.Sprintf("%.2f", stats.ContentionWrite)),
		"",
		m.styles.label.Render("File Cache"),
		m.metricLine("Max Size", fmt.Sprintf("%d", stats.CacheFilesMax), "Occupancy", fmt.Sprintf("%d", stats.CacheFilesOccup)),
		m.metricLine("Requests", fmt.Sprintf("%d", stats.CacheFilesRequests), "Hits", fmt.Sprintf("%d", stats.CacheFilesHits)),
		"",
		m.styles.label.Render("Container Cache"),
		m.metricLine("Max Size", fmt.Sprintf("%d", stats.CacheContainersMax), "Occupancy", fmt.Sprintf("%d", stats.CacheContainersOccup)),
		m.metricLine("Requests", fmt.Sprintf("%d", stats.CacheContainersRequests), "Hits", fmt.Sprintf("%d", stats.CacheContainersHits)),
	}

	return m.styles.panelDim.Width(contentWidth).Render(fitLines(lines, height))
}

func (m model) renderSpaceStatusView(height int) string {
	width := m.contentWidth()
	contentWidth := panelContentWidth(width)

	if m.spaceStatusLoading {
		return m.styles.panelDim.Width(width).Render(fitLines([]string{"Loading space status..."}, height))
	}
	if m.spaceStatusErr != nil {
		return m.styles.panelDim.Width(width).Render(fitLines([]string{m.styles.error.Render(m.spaceStatusErr.Error())}, height))
	}

	columns := allocateTableColumns(contentWidth, []tableColumn{
		{title: "parameter", min: 36, weight: 1},
		{title: "value", min: 40, weight: 2},
	})

	title := m.styles.label.Render("EOS Space Status (default)")
	lines := []string{
		title,
		"",
		m.renderSimpleHeaderRow(columns, []string{"parameter", "value"}),
	}

	start, end := visibleWindow(len(m.spaceStatus), m.spaceStatusSelected, max(1, height-len(lines)))
	for i := start; i < end; i++ {
		record := m.spaceStatus[i]
		line := formatTableRow(columns, []string{record.Key, record.Value})
		if i == m.spaceStatusSelected {
			line = m.styles.selected.Width(contentWidth).Render(line)
		}
		lines = append(lines, line)
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, height))
}

type ioShapingMergedRow struct {
	id      string
	traffic *eos.IOShapingRecord
	policy  *eos.IOShapingPolicyRecord
}

// ioShapingMergedRows returns the union of traffic records and policy records
// for the current mode, sorted alphabetically by id. Rows with traffic but no
// policy, policy but no traffic, or both are all included.
func (m model) ioShapingMergedRows() []ioShapingMergedRow {
	policyType := "app"
	switch m.ioShapingMode {
	case eos.IOShapingUsers:
		policyType = "uid"
	case eos.IOShapingGroups:
		policyType = "gid"
	}

	policyByID := make(map[string]eos.IOShapingPolicyRecord)
	for _, p := range m.ioShapingPolicies {
		if strings.ToLower(p.Type) == policyType {
			policyByID[p.ID] = p
		}
	}

	seen := make(map[string]bool)
	var rows []ioShapingMergedRow
	for i := range m.ioShaping {
		r := &m.ioShaping[i]
		seen[r.ID] = true
		row := ioShapingMergedRow{id: r.ID, traffic: r}
		if p, ok := policyByID[r.ID]; ok {
			row.policy = &p
		}
		rows = append(rows, row)
	}
	for id, p := range policyByID {
		if !seen[id] {
			pc := p
			rows = append(rows, ioShapingMergedRow{id: id, policy: &pc})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].id < rows[j].id })
	return rows
}

func (m model) renderIOShapingView(height int) string {
	width := m.contentWidth()
	contentWidth := panelContentWidth(width)

	idLabel := "application"
	switch m.ioShapingMode {
	case eos.IOShapingUsers:
		idLabel = "uid"
	case eos.IOShapingGroups:
		idLabel = "gid"
	}

	indicator := ""
	if m.ioShapingLoading {
		indicator = m.styles.status.Render("  ↻")
	}

	if m.ioShapingErr != nil {
		lines := []string{
			m.styles.label.Render("IO Traffic Shaping") + indicator,
			"",
			m.styles.error.Render(m.ioShapingErr.Error()),
		}
		return m.styles.panelDim.Width(width).Render(fitLines(lines, height))
	}

	rows := m.ioShapingMergedRows()

	formatLimit := func(v float64) string {
		if v == 0 {
			return "-"
		}
		return humanBytesRate(v)
	}
	enabledStr := func(p *eos.IOShapingPolicyRecord) string {
		if p == nil {
			return "-"
		}
		if p.Enabled {
			return "yes"
		}
		return "no"
	}

	dataRows := make([][]string, len(rows))
	for i, r := range rows {
		readRate, writeRate, readIOPS, writeIOPS := "-", "-", "-", "-"
		if r.traffic != nil {
			readRate = humanBytesRate(r.traffic.ReadBps)
			writeRate = humanBytesRate(r.traffic.WriteBps)
			readIOPS = fmt.Sprintf("%.1f", r.traffic.ReadIOPS)
			writeIOPS = fmt.Sprintf("%.1f", r.traffic.WriteIOPS)
		}
		limRead, limWrite, resRead, resWrite := "-", "-", "-", "-"
		if r.policy != nil {
			limRead = formatLimit(r.policy.LimitReadBytesPerSec)
			limWrite = formatLimit(r.policy.LimitWriteBytesPerSec)
			resRead = formatLimit(r.policy.ReservationReadBytesPerSec)
			resWrite = formatLimit(r.policy.ReservationWriteBytesPerSec)
		}
		dataRows[i] = []string{
			r.id,
			readRate, writeRate, readIOPS, writeIOPS,
			enabledStr(r.policy),
			limRead, limWrite, resRead, resWrite,
		}
	}

	headers := []string{idLabel, "read rate", "write rate", "read iops", "write iops", "enabled", "lim read", "lim write", "res read", "res write"}
	columns := allocateTableColumns(contentWidth, contentAwareColumns([]tableColumn{
		{title: idLabel, min: 10, weight: 4},
		{title: "read rate", min: 10, weight: 1, right: true},
		{title: "write rate", min: 10, weight: 1, right: true},
		{title: "read iops", min: 9, weight: 0, right: true},
		{title: "write iops", min: 10, weight: 0, right: true},
		{title: "enabled", min: 7, weight: 0},
		{title: "lim read", min: 10, weight: 0, right: true},
		{title: "lim write", min: 10, weight: 0, right: true},
		{title: "res read", min: 10, weight: 0, right: true},
		{title: "res write", min: 10, weight: 0, right: true},
	}, dataRows))

	title := m.styles.label.Render("IO Traffic  ") +
		m.styles.label.Render("5s window  ") +
		modeTabLabel(m.ioShapingMode, eos.IOShapingApps, "a apps", m.styles) + "  " +
		modeTabLabel(m.ioShapingMode, eos.IOShapingUsers, "u users", m.styles) + "  " +
		modeTabLabel(m.ioShapingMode, eos.IOShapingGroups, "g groups", m.styles) +
		indicator

	lines := []string{title, "", m.renderSimpleHeaderRow(columns, headers)}

	if m.ioShapingLoading && len(rows) == 0 {
		lines = append(lines, "Loading...")
	} else if len(rows) == 0 {
		lines = append(lines, "(no data)")
	} else {
		start, end := visibleWindow(len(rows), m.ioShapingSelected, max(1, height-len(lines)))
		lines[0] = title + renderScrollSummary(start, end, len(rows))
		for i := start; i < end; i++ {
			line := formatTableRow(columns, dataRows[i])
			if i == m.ioShapingSelected {
				line = m.styles.selected.Width(contentWidth).Render(line)
			}
			lines = append(lines, line)
		}
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, height))
}

func modeTabLabel(current, target eos.IOShapingMode, label string, s styles) string {
	if current == target {
		return s.tabActive.Render(label)
	}
	return s.tab.Render(label)
}

func humanBytesRate(bps float64) string {
	switch {
	case bps >= 1e9:
		return fmt.Sprintf("%.2f GB/s", bps/1e9)
	case bps >= 1e6:
		return fmt.Sprintf("%.2f MB/s", bps/1e6)
	case bps >= 1e3:
		return fmt.Sprintf("%.2f KB/s", bps/1e3)
	default:
		return fmt.Sprintf("%.0f B/s", bps)
	}
}

func (m model) renderSpaceStatusEditPopup() string {
	cancelBtn := "[ Cancel ]"
	continueBtn := "[ Continue ]"

	if !m.edit.focusInput {
		if m.edit.button == buttonCancel {
			cancelBtn = m.styles.selected.Render(cancelBtn)
		} else {
			continueBtn = m.styles.selected.Render(continueBtn)
		}
	}

	lines := []string{
		m.styles.label.Render("Edit Space Status"),
		fmt.Sprintf("Key:   %s", m.styles.value.Render(m.edit.record.Key)),
		fmt.Sprintf("Value: %s", m.styles.value.Render(m.edit.record.Value)),
		"",
		m.edit.input.View(),
		"",
		lipgloss.JoinHorizontal(lipgloss.Left, cancelBtn, "  ", continueBtn),
	}

	return m.styles.panel.
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m model) renderSpaceStatusConfirmPopup() string {
	cancelBtn := "[ Cancel ]"
	confirmBtn := "[ Confirm ]"

	if m.edit.button == buttonCancel {
		cancelBtn = m.styles.selected.Render(cancelBtn)
	} else {
		confirmBtn = m.styles.selected.Render(confirmBtn)
	}

	command := fmt.Sprintf("eos space config default %s=%s", m.edit.record.Key, m.edit.input.Value())

	lines := []string{
		m.styles.label.Render("Confirm Configuration Change"),
		"",
		"The following command will be executed:",
		"",
		m.styles.value.Render(command),
		"",
		lipgloss.JoinHorizontal(lipgloss.Left, cancelBtn, "  ", confirmBtn),
	}

	return m.styles.panel.
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m model) renderNamespaceView(height int) string {
	width := m.contentWidth()

	fixedListLines := 3 // Title, blank, header
	naturalListContent := fixedListLines + len(m.directory.Entries)
	if !m.nsLoaded && !m.nsLoading {
		naturalListContent = 4 // Title, blank, header, "(empty)" or hint
	}

	// Details have dynamic height: 7 base lines + 2 for container/file info + optional link line.
	naturalDetailContent := 9
	if selected, ok := m.selectedNamespaceEntry(); ok {
		if selected.Kind != eos.EntryKindContainer {
			if selected.LinkName != "" {
				naturalDetailContent = 10
			}
		}
	} else if m.directory.Self.Kind != eos.EntryKindContainer {
		if m.directory.Self.LinkName != "" {
			naturalDetailContent = 10
		}
	}

	listHeight, detailHeight := adaptiveSplitHeights(height, naturalListContent, naturalDetailContent)

	list := m.renderNamespaceList(width, listHeight)
	details := m.renderNamespaceDetails(width, detailHeight)
	return lipgloss.JoinVertical(lipgloss.Left, list, details)
}

func (m model) renderNamespaceList(width, height int) string {
	contentWidth := panelContentWidth(width)
	columns := allocateTableColumns(contentWidth, []tableColumn{
		{title: "type", min: 4, weight: 1},
		{title: "name", min: 24, weight: 6},
		{title: "size", min: 10, weight: 2, right: true},
		{title: "uid", min: 6, weight: 1, right: true},
		{title: "gid", min: 6, weight: 1, right: true},
		{title: "modified", min: 16, weight: 2},
	})

	title := m.styles.label.Render("Namespace Path ") + m.styles.value.Render(m.directory.Path)
	lines := []string{
		title,
		"",
		m.renderNamespaceHeaderRow(columns),
	}

	if m.nsLoading {
		lines = append(lines, "Loading directory listing...")
	} else if m.nsErr != nil {
		lines = append(lines, m.styles.error.Render(m.nsErr.Error()))
	} else if len(m.directory.Entries) == 0 {
		lines = append(lines, "(empty)")
	} else {
		start, end := visibleWindow(len(m.directory.Entries), m.nsSelected, max(1, panelContentHeight(height)-len(lines)))
		lines[0] = title + renderScrollSummary(start, end, len(m.directory.Entries))
		for i := start; i < end; i++ {
			entry := m.directory.Entries[i]
			line := formatTableRow(columns, []string{
				entryTypeLabel(entry),
				entry.Name,
				entrySize(entry),
				fmt.Sprintf("%d", entry.UID),
				fmt.Sprintf("%d", entry.GID),
				formatTimeShort(entry.ModifiedAt),
			})
			if i == m.nsSelected {
				line = m.styles.selected.Width(contentWidth).Render(line)
			}
			lines = append(lines, line)
		}
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, panelContentHeight(height)))
}

func (m model) renderNamespaceDetails(width, height int) string {
	target := m.directory.Self
	if selected, ok := m.selectedNamespaceEntry(); ok {
		target = selected
	}

	lines := []string{
		m.styles.label.Render("Selected Namespace Entry"),
		truncate(target.Path, max(10, width-4)),
		"",
		m.metricLine("Type", entryTypeLabel(target), "ID", fmt.Sprintf("%d", target.ID)),
		m.metricLine("UID", fmt.Sprintf("%d", target.UID), "GID", fmt.Sprintf("%d", target.GID)),
		m.metricLine("Size", entrySize(target), "Inode", fmt.Sprintf("%d", target.Inode)),
		m.metricLine("Modified", formatTime(target.ModifiedAt), "Changed", formatTime(target.ChangedAt)),
	}

	if target.Kind == eos.EntryKindContainer {
		lines = append(lines,
			m.metricLine("Tree Files", fmt.Sprintf("%d", target.Files), "Tree Dirs", fmt.Sprintf("%d", target.Containers)),
			m.metricLine("Tree Size", humanBytes(uint64(max64(target.TreeSize, 0))), "Mode", fmt.Sprintf("0%o", target.Mode)),
		)
	} else {
		lines = append(lines,
			m.metricLine("Layout", fmt.Sprintf("%d", target.LayoutID), "Locations", fmt.Sprintf("%d", target.Locations)),
			m.metricLine("Flags", fmt.Sprintf("0x%x", target.Flags), "ETag", fallback(target.ETag, "-")),
		)
		if target.LinkName != "" {
			lines = append(lines, m.metricLine("Link", target.LinkName, "", ""))
		}
	}

	return m.styles.panelDim.Width(width).Render(fitLines(lines, panelContentHeight(height)))
}

func (m model) renderFooter() string {
	if m.log.active {
		filter := ""
		if m.log.filter != "" {
			filter = fmt.Sprintf("  •  filter: %q", m.log.filter)
		}
		keys := fmt.Sprintf("↑↓/jk scroll  •  g top  •  G bottom  •  / filter  •  r reload  •  esc close%s", filter)
		if m.log.filtering {
			keys = "type to filter  •  enter apply  •  esc cancel"
		}
		return m.styles.status.Width(m.contentWidth()).Render(keys)
	}

	hostViews := m.activeView == viewMGM || m.activeView == viewQDB ||
		m.activeView == viewFST || m.activeView == viewFileSystems
	var keys string
	switch m.activeView {
	case viewNamespace:
		keys = "tab/1-0 switch  •  ↑↓/jk scroll  •  ctrl+d/u half-page  •  ←→ navigate  •  enter open  •  backspace back  •  g root  •  q quit"
	case viewIOShaping:
		keys = "tab/1-0 switch  •  ↑↓/jk scroll  •  ctrl+d/u half-page  •  a apps  •  u users  •  g groups  •  r refresh  •  q quit"
	case viewFileSystems:
		keys = "tab/1-0 switch  •  ↑↓/jk scroll  •  ctrl+d/u half-page  •  ←→ column  •  S sort  •  /filter  •  enter edit configstatus  •  l logs  •  s shell  •  q quit"
	default:
		keys = "tab/1-0 switch  •  ↑↓/jk scroll  •  ctrl+d/u half-page  •  ←→ column  •  S sort  •  /filter  •  q quit"
		if hostViews {
			keys = "tab/1-0 switch  •  ↑↓/jk scroll  •  ctrl+d/u half-page  •  ←→ column  •  S sort  •  /filter  •  l logs  •  s shell  •  q quit"
		}
	}

	return m.styles.status.Width(m.contentWidth()).Render(keys)
}

func (m model) metricLine(leftLabel, leftValue, rightLabel, rightValue string) string {
	left := m.styles.label.Render(leftLabel+" ") + m.styles.value.Render(leftValue)
	if rightLabel == "" {
		return left
	}

	right := m.styles.label.Render(rightLabel+" ") + m.styles.value.Render(rightValue)
	return fmt.Sprintf("%-42s %s", left, right)
}

func (m model) contentWidth() int {
	return max(20, m.width-2)
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

// computeClusterHealth derives a health state string from already-loaded node
// and filesystem data. This avoids calling `eos status` (which internally runs
// the eos-status shell script and creates temporary files).
// Returns "OK", "WARN", or "-" (if no data yet).
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

func (m model) visibleFSTs() []eos.FstRecord {
	fsts := make([]eos.FstRecord, 0, len(m.fsts))
	for _, node := range m.fsts {
		t := strings.ToLower(node.Type)
		// A node is an FST if it's explicitly type 'fst' OR if it has registered filesystems.
		if t == "fst" || node.FileSystemCount > 0 {
			fsts = append(fsts, node)
		}
	}

	if len(m.fstFilter.filters) > 0 {
		filtered := make([]eos.FstRecord, 0, len(fsts))
		for _, node := range fsts {
			if m.matchesNodeFilters(node) {
				filtered = append(filtered, node)
			}
		}
		fsts = filtered
	}
	if m.fstSort.column >= 0 {
		sort.SliceStable(fsts, func(i, j int) bool {
			return m.lessNode(fsts[i], fsts[j])
		})
	}
	return fsts
}

func (m model) visibleFileSystems() []eos.FileSystemRecord {
	fileSystems := append([]eos.FileSystemRecord(nil), m.fileSystems...)
	if len(m.fsFilter.filters) > 0 {
		filtered := make([]eos.FileSystemRecord, 0, len(fileSystems))
		for _, fs := range fileSystems {
			if m.matchesFileSystemFilters(fs) {
				filtered = append(filtered, fs)
			}
		}
		fileSystems = filtered
	}
	if m.fsSort.column >= 0 {
		sort.SliceStable(fileSystems, func(i, j int) bool {
			return m.lessFileSystem(fileSystems[i], fileSystems[j])
		})
	}
	return fileSystems
}

func (m model) fstFilterValue(node eos.FstRecord) string {
	return m.fstFilterValueForColumn(node, m.fstFilter.column)
}

func (m model) fstFilterValueForColumn(node eos.FstRecord, column int) string {
	switch fstFilterColumn(column) {
	case fstFilterHost:
		return node.Host
	case fstFilterPort:
		return fmt.Sprintf("%d", node.Port)
	case fstFilterGeotag:
		return node.Geotag
	case fstFilterStatus:
		return node.Status
	case fstFilterActivated:
		return node.Activated
	case fstFilterHeartbeatDelta:
		return fmt.Sprintf("%d", node.HeartbeatDelta)
	case fstFilterNoFS:
		return fmt.Sprintf("%d", node.FileSystemCount)
	case fstFilterEOSVersion:
		return node.EOSVersion
	case fstFilterType:
		return node.Type
	default:
		return node.Host
	}
}

func (m model) fsFilterValue(fs eos.FileSystemRecord) string {
	return m.fsFilterValueForColumn(fs, m.fsFilter.column)
}

func (m model) fsFilterValueForColumn(fs eos.FileSystemRecord, column int) string {
	switch fsFilterColumn(column) {
	case fsFilterPort:
		return fmt.Sprintf("%d", fs.Port)
	case fsFilterID:
		return fmt.Sprintf("%d", fs.ID)
	case fsFilterPath:
		return fs.Path
	case fsFilterGroup:
		return fs.SchedGroup
	case fsFilterGeotag:
		return fs.Geotag
	case fsFilterBoot:
		return fs.Boot
	case fsFilterConfigStatus:
		return fs.ConfigStatus
	case fsFilterDrain:
		return fs.DrainStatus
	case fsFilterUsage:
		return fmt.Sprintf("%.2f", usagePercent(fs.UsedBytes, fs.CapacityBytes))
	case fsFilterStatus:
		return fs.Active
	case fsFilterHealth:
		return fs.Health
	default:
		return fs.Host
	}
}

func (m model) matchesNodeFilters(node eos.FstRecord) bool {
	for column, query := range m.fstFilter.filters {
		if query == "" {
			continue
		}
		value := strings.ToLower(m.fstFilterValueForColumn(node, column))
		if !strings.Contains(value, strings.ToLower(query)) {
			return false
		}
	}
	return true
}

func (m model) matchesFileSystemFilters(fs eos.FileSystemRecord) bool {
	for column, query := range m.fsFilter.filters {
		if query == "" {
			continue
		}
		value := strings.ToLower(m.fsFilterValueForColumn(fs, column))
		if !strings.Contains(value, strings.ToLower(query)) {
			return false
		}
	}
	return true
}

func (m model) lessNode(a, b eos.FstRecord) bool {
	var less bool
	switch fstSortColumn(m.fstSort.column) {
	case fstSortType:
		less = strings.Compare(a.Type, b.Type) < 0
	case fstSortHost:
		less = strings.Compare(a.Host, b.Host) < 0
	case fstSortPort:
		less = a.Port < b.Port
	case fstSortStatus:
		less = strings.Compare(a.Status, b.Status) < 0
	case fstSortGeotag:
		less = strings.Compare(a.Geotag, b.Geotag) < 0
	case fstSortActivated:
		less = strings.Compare(a.Activated, b.Activated) < 0
	case fstSortNoFS:
		less = a.FileSystemCount < b.FileSystemCount
	case fstSortHeartbeat:
		less = a.HeartbeatDelta < b.HeartbeatDelta
	case fstSortEOSVersion:
		less = strings.Compare(a.EOSVersion, b.EOSVersion) < 0
	default:
		less = strings.Compare(a.Host, b.Host) < 0
	}
	if equivalentNodeSortValue(m.fstSort.column, a, b) {
		less = strings.Compare(a.Host, b.Host) < 0
	}
	if m.fstSort.desc {
		return !less
	}
	return less
}

func (m model) lessFileSystem(a, b eos.FileSystemRecord) bool {
	var less bool
	switch fsSortColumn(m.fsSort.column) {
	case fsSortHost:
		less = strings.Compare(a.Host, b.Host) < 0
	case fsSortPort:
		less = a.Port < b.Port
	case fsSortID:
		less = a.ID < b.ID
	case fsSortPath:
		less = strings.Compare(a.Path, b.Path) < 0
	case fsSortGroup:
		less = strings.Compare(a.SchedGroup, b.SchedGroup) < 0
	case fsSortGeotag:
		less = strings.Compare(a.Geotag, b.Geotag) < 0
	case fsSortBoot:
		less = strings.Compare(a.Boot, b.Boot) < 0
	case fsSortConfigStatus:
		less = strings.Compare(a.ConfigStatus, b.ConfigStatus) < 0
	case fsSortDrain:
		less = strings.Compare(a.DrainStatus, b.DrainStatus) < 0
	case fsSortUsed:
		less = usagePercent(a.UsedBytes, a.CapacityBytes) < usagePercent(b.UsedBytes, b.CapacityBytes)
	case fsSortStatus:
		less = strings.Compare(a.Active, b.Active) < 0
	case fsSortHealth:
		less = strings.Compare(a.Health, b.Health) < 0
	default:
		less = a.ID < b.ID
	}
	if equivalentFileSystemSortValue(m.fsSort.column, a, b) {
		less = a.ID < b.ID
	}
	if m.fsSort.desc {
		return !less
	}
	return less
}

func (m model) renderNodeControls() string {
	return fmt.Sprintf("  [col:%s filters:%d current:%s]",
		m.fstSelectedColumnLabel(),
		len(m.fstFilter.filters),
		filterValueLabel(m.fstFilter.filters[m.fstColumnSelected], m.popup.active && m.popup.view == viewFST, m.popup.input.Value()),
	)
}

func (m model) renderFileSystemControls() string {
	return fmt.Sprintf("  [col:%s filters:%d current:%s]",
		m.fsSelectedColumnLabel(),
		len(m.fsFilter.filters),
		filterValueLabel(m.fsFilter.filters[m.fsColumnSelected], m.popup.active && m.popup.view == viewFileSystems, m.popup.input.Value()),
	)
}

func filterValueLabel(current string, active bool, input string) string {
	if active {
		return fmt.Sprintf("%q*", input)
	}
	if current == "" {
		return "\"\""
	}
	return fmt.Sprintf("%q", current)
}

func padVisibleWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	w := lipgloss.Width(s)
	if w >= width {
		return ansi.Cut(s, 0, width)
	}
	return s + strings.Repeat(" ", width-w)
}

func sortDirectionLabel(desc bool) string {
	if desc {
		return "desc"
	}
	return "asc"
}

// NOTE: When adding or removing columns, ensure the labels slice here
// remains in exact sync with the dataRows in renderNodesList (renderFstView).
func (m model) renderFstHeaderRow(columns []tableColumn) string {
	labels := []string{"host", "port", "geotag", "status", "activated", "heartbeatdelta", "nofs", "eos version"}
	return m.renderSelectableHeaderRow(columns, labels, m.fstColumnSelected, m.fstSort, m.fstFilter)
}

func (m model) renderFileSystemHeaderRow(columns []tableColumn) string {
	labels := []string{"host", "port", "id", "path", "schedgroup", "geotag", "boot", "configstatus", "drain", "usage %", "active", "health"}
	return m.renderSelectableHeaderRow(columns, labels, m.fsColumnSelected, m.fsSort, m.fsFilter)
}

// renderSimpleHeaderRow renders a plain (non-selectable, non-sortable) column
// header row using the label style.  ALL column headers MUST go through either
// this function, renderSelectableHeaderRow, or a view-specific wrapper that
// calls one of those two — never m.styles.header (that is for the app title bar
// only).
func (m model) renderSimpleHeaderRow(columns []tableColumn, labels []string) string {
	cells := make([]string, len(columns))
	for i, col := range columns {
		label := ""
		if i < len(labels) {
			label = labels[i]
		}
		var cell string
		if col.right {
			cell = padLeft(label, col.min)
		} else {
			cell = padRight(label, col.min)
		}
		cells[i] = m.styles.label.Render(cell)
	}
	return strings.Join(cells, " ")
}

func (m model) renderNamespaceHeaderRow(columns []tableColumn) string {
	return m.renderSimpleHeaderRow(columns, []string{"type", "name", "size", "uid", "gid", "modified"})
}

func (m model) renderSelectableHeaderRow(columns []tableColumn, labels []string, selected int, sortState sortState, filterState filterState) string {
	cells := make([]string, 0, len(columns))
	for i, column := range columns {
		label := ""
		if i < len(labels) {
			label = labels[i]
		}
		if sortState.column == i {
			if sortState.desc {
				label += "↓"
			} else {
				label += "↑"
			}
		}
		if filterState.filters[i] != "" {
			label += "*"
		}
		if i == selected {
			label = "[" + label + "]"
		}
		cell := padRight(label, column.min)
		if i == selected {
			cell = m.styles.selected.Render(cell)
		} else {
			cell = m.styles.label.Render(cell)
		}
		cells = append(cells, cell)
	}
	return strings.Join(cells, " ")
}

// renderFilterSummary returns a line showing all active filters (for display
// below the column header row).  labelFn maps column index → label string.
func (m model) renderFilterSummary(filters map[int]string, labelFn func(int) string) string {
	cols := make([]int, 0, len(filters))
	for col, v := range filters {
		if v != "" {
			cols = append(cols, col)
		}
	}
	if len(cols) == 0 {
		return ""
	}
	sort.Ints(cols)
	parts := make([]string, 0, len(cols))
	for _, col := range cols {
		parts = append(parts, m.styles.label.Render(labelFn(col)+"=")+m.styles.value.Render(filters[col]))
	}
	return m.styles.label.Render("active filters: ") + strings.Join(parts, m.styles.status.Render("  •  "))
}

func (m model) renderFilterPopup() string {
	title := "Filter " + m.activeFilterColumnLabel()
	if m.popup.view == viewFileSystems {
		title = "Filter " + m.fsFilterColumnLabel()
	}

	contentWidth := min(80, max(40, m.contentWidth()-8))
	inputView := m.popup.input.View()
	tableView := m.popup.table.View()
	hint := m.styles.status.Render("Enter apply selected value • Esc cancel")

	box := lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.header.Render(title),
		"",
		inputView,
		"",
		tableView,
		"",
		hint,
	)

	return m.styles.panelDim.Width(contentWidth).Render(box)
}

func (m model) fstFilterColumnLabel() string {
	switch fstFilterColumn(m.fstFilter.column) {
	case fstFilterHost:
		return "host"
	case fstFilterPort:
		return "port"
	case fstFilterGeotag:
		return "geotag"
	case fstFilterStatus:
		return "status"
	case fstFilterActivated:
		return "activated"
	case fstFilterHeartbeatDelta:
		return "heartbeatdelta"
	case fstFilterNoFS:
		return "nofs"
	case fstFilterEOSVersion:
		return "eos version"
	case fstFilterType:
		return "type"
	default:
		return "host"
	}
}

func (m model) fstSortColumnLabel() string {
	switch fstSortColumn(m.fstSort.column) {
	case fstSortHost:
		return "host"
	case fstSortPort:
		return "port"
	case fstSortGeotag:
		return "geotag"
	case fstSortStatus:
		return "status"
	case fstSortActivated:
		return "activated"
	case fstSortHeartbeat:
		return "heartbeatdelta"
	case fstSortNoFS:
		return "nofs"
	case fstSortEOSVersion:
		return "eos version"
	case fstSortType:
		return "type"
	case fstSortNone:
		return "none"
	default:
		return "host"
	}
}

func (m model) fsFilterColumnLabel() string {
	switch fsFilterColumn(m.fsFilter.column) {
	case fsFilterPort:
		return "port"
	case fsFilterID:
		return "id"
	case fsFilterPath:
		return "path"
	case fsFilterGroup:
		return "schedgroup"
	case fsFilterGeotag:
		return "geotag"
	case fsFilterBoot:
		return "boot"
	case fsFilterConfigStatus:
		return "configstatus"
	case fsFilterDrain:
		return "drain"
	case fsFilterUsage:
		return "usage %"
	case fsFilterStatus:
		return "active"
	case fsFilterHealth:
		return "health"
	default:
		return "host"
	}
}

func (m model) fsSortColumnLabel() string {
	switch fsSortColumn(m.fsSort.column) {
	case fsSortHost:
		return "host"
	case fsSortPort:
		return "port"
	case fsSortID:
		return "id"
	case fsSortPath:
		return "path"
	case fsSortGroup:
		return "schedgroup"
	case fsSortGeotag:
		return "geotag"
	case fsSortBoot:
		return "boot"
	case fsSortConfigStatus:
		return "configstatus"
	case fsSortDrain:
		return "drain"
	case fsSortUsed:
		return "usage %"
	case fsSortStatus:
		return "active"
	case fsSortHealth:
		return "health"
	case fsSortNone:
		return "none"
	default:
		return "host"
	}
}

func (m model) activeFilterColumnLabel() string {
	switch m.activeView {
	case viewFileSystems:
		return m.fsFilterColumnLabel()
	case viewGroups:
		return m.groupFilterColumnLabel()
	default:
		return m.fstFilterColumnLabel()
	}
}

func (m *model) openFilterPopup() {
	m.popup.active = true
	m.popup.view = m.activeView
	if m.activeView == viewFileSystems {
		m.popup.column = m.fsColumnSelected
		m.popup.input.SetValue(m.fsFilter.filters[m.fsColumnSelected])
	} else if m.activeView == viewGroups {
		m.popup.column = m.groupsColumnSelected
		m.popup.input.SetValue(m.groupFilter.filters[m.groupsColumnSelected])
	} else {
		m.popup.column = m.fstColumnSelected
		m.popup.input.SetValue(m.fstFilter.filters[m.fstColumnSelected])
	}
	m.popup.input.CursorEnd()
	m.popup.input.Focus()
	m.popup.table.Focus()
	// Populate rows immediately so the table is ready for keyboard navigation
	// without requiring a text-input event first.
	m.updatePopupRows()
	m.popup.table.SetCursor(0)
	m.status = fmt.Sprintf("Select filter for %s", m.activeFilterColumnLabel())
}

func (m *model) closeFilterPopup(status string) {
	m.popup.active = false
	m.popup.input.Blur()
	m.popup.input.SetValue("")
	m.popup.values = nil
	m.popup.table.SetRows(nil)
	m.status = status
}

func (m *model) applyPopupSelection() {
	row := m.popup.table.SelectedRow()
	if len(row) == 0 {
		m.closeFilterPopup("No filter value selected")
		return
	}

	value := row[0]
	if value == "(no matches)" {
		m.closeFilterPopup("No matching filter value")
		return
	}
	if value == "(no filter)" {
		value = ""
	}

	switch m.popup.view {
	case viewFileSystems:
		m.fsFilter.column = m.popup.column
		if value == "" {
			delete(m.fsFilter.filters, m.popup.column)
		} else {
			m.fsFilter.filters[m.popup.column] = value
		}
		m.fsSelected = clampIndex(0, len(m.visibleFileSystems()))
		m.closeFilterPopup(fmt.Sprintf("Filesystem filters active: %d", len(m.fsFilter.filters)))
	case viewGroups:
		m.groupFilter.column = m.popup.column
		if value == "" {
			delete(m.groupFilter.filters, m.popup.column)
		} else {
			m.groupFilter.filters[m.popup.column] = value
		}
		m.groupsSelected = clampIndex(0, len(m.visibleGroups()))
		m.closeFilterPopup(fmt.Sprintf("Group filters active: %d", len(m.groupFilter.filters)))
	default:
		m.fstFilter.column = m.popup.column
		if value == "" {
			delete(m.fstFilter.filters, m.popup.column)
		} else {
			m.fstFilter.filters[m.popup.column] = value
		}
		m.fstSelected = clampIndex(0, len(m.visibleFSTs()))
		m.closeFilterPopup(fmt.Sprintf("Node filters active: %d", len(m.fstFilter.filters)))
	}
}

func (m *model) updatePopupRows() {
	needle := strings.ToLower(strings.TrimSpace(m.popup.input.Value()))
	values := m.popupValues()
	rows := make([]table.Row, 0, len(values))
	for _, value := range values {
		label := value
		if label == "" {
			label = "(no filter)"
		}
		if needle == "" || strings.Contains(strings.ToLower(label), needle) {
			rows = append(rows, table.Row{label})
		}
	}
	if len(rows) == 0 {
		rows = []table.Row{{"(no matches)"}}
	}
	m.popup.table.SetColumns([]table.Column{{Title: "value", Width: min(60, max(24, m.contentWidth()-16))}})
	m.popup.table.SetRows(rows)
	m.popup.table.SetHeight(min(14, max(6, m.height/3)))
	m.popup.table.SetWidth(min(70, max(28, m.contentWidth()-12)))
	m.popup.table.SetCursor(0)
}

func (m model) popupValues() []string {
	values := []string{""}
	seen := map[string]bool{"": true}
	// Only show values that pass all *other* active filters so the list stays
	// consistent with what the user would actually see after applying the filter.
	switch m.popup.view {
	case viewFileSystems:
		for _, fs := range m.fileSystems {
			if !m.matchesFileSystemFiltersExcept(fs, m.popup.column) {
				continue
			}
			value := m.fsFilterValueForColumn(fs, m.popup.column)
			if !seen[value] {
				seen[value] = true
				values = append(values, value)
			}
		}
	default:
		for _, node := range m.fsts {
			if !m.matchesNodeFiltersExcept(node, m.popup.column) {
				continue
			}
			value := m.fstFilterValueForColumn(node, m.popup.column)
			if !seen[value] {
				seen[value] = true
				values = append(values, value)
			}
		}
	}
	sort.Strings(values[1:])
	return values
}

func (m model) matchesNodeFiltersExcept(node eos.FstRecord, excludeColumn int) bool {
	for col, query := range m.fstFilter.filters {
		if col == excludeColumn || query == "" {
			continue
		}
		if !strings.Contains(strings.ToLower(m.fstFilterValueForColumn(node, col)), strings.ToLower(query)) {
			return false
		}
	}
	return true
}

func (m model) matchesFileSystemFiltersExcept(fs eos.FileSystemRecord, excludeColumn int) bool {
	for col, query := range m.fsFilter.filters {
		if col == excludeColumn || query == "" {
			continue
		}
		if !strings.Contains(strings.ToLower(m.fsFilterValueForColumn(fs, col)), strings.ToLower(query)) {
			return false
		}
	}
	return true
}

func nodeColumnCount() int {
	return 8 // the 8 navigable visible columns; fstFilterType/fstSortType are not user-navigable
}

func fsColumnCount() int {
	return 12
}

func (m model) fstSelectedColumnLabel() string {
	column := m.fstFilter.column
	m.fstFilter.column = m.fstColumnSelected
	label := m.fstFilterColumnLabel()
	m.fstFilter.column = column
	return label
}

func (m model) fsSelectedColumnLabel() string {
	column := m.fsFilter.column
	m.fsFilter.column = m.fsColumnSelected
	label := m.fsFilterColumnLabel()
	m.fsFilter.column = column
	return label
}

func (m model) fstSortStateLabel() string {
	if m.fstSort.column < 0 {
		return "none"
	}
	return fmt.Sprintf("%s %s", m.fstSortColumnLabel(), sortDirectionLabel(m.fstSort.desc))
}

func (m model) fsSortStateLabel() string {
	if m.fsSort.column < 0 {
		return "none"
	}
	return fmt.Sprintf("%s %s", m.fsSortColumnLabel(), sortDirectionLabel(m.fsSort.desc))
}

func (m model) nextNodeSortState() sortState {
	selected := m.fstColumnSelected
	if m.fstSort.column != selected {
		return sortState{column: selected}
	}
	if !m.fstSort.desc {
		return sortState{column: selected, desc: true}
	}
	return sortState{column: int(fstSortNone)}
}

func (m model) nextFileSystemSortState() sortState {
	selected := m.fsColumnSelected
	if m.fsSort.column != selected {
		return sortState{column: selected}
	}
	if !m.fsSort.desc {
		return sortState{column: selected, desc: true}
	}
	return sortState{column: int(fsSortNone)}
}

func (m model) nodeColumnIsEnum(column int) bool {
	switch fstFilterColumn(column) {
	case fstFilterType, fstFilterStatus, fstFilterActivated:
		return true
	default:
		return false
	}
}

func (m model) fsColumnIsEnum(column int) bool {
	switch fsFilterColumn(column) {
	case fsFilterBoot, fsFilterConfigStatus, fsFilterDrain, fsFilterStatus:
		return true
	default:
		return false
	}
}

// loadInfraCmd fans out all infrastructure fetches in parallel.  Each
// component delivers its own typed message to the Bubble Tea runtime as soon
// as it completes, so a slow or timing-out command (e.g. NodeStats) never
// delays the display of faster data (e.g. FST node list).
func loadInfraCmd(c *eos.Client) tea.Cmd {
	return tea.Batch(
		loadNodeStatsCmd(c),
		loadFSTsCmd(c),
		loadMGMsCmd(c),
		loadFileSystemsCmd(c),
		loadEOSVersionCmd(c),
		loadSpacesCmd(c),
		loadNamespaceStatsCmd(c),
	)
}

func loadNodeStatsCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		stats, err := client.NodeStats(context.Background())
		return nodeStatsLoadedMsg{stats: stats, err: err}
	}
}

func loadFSTsCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		fsts, err := client.Nodes(context.Background())
		return fstsLoadedMsg{fsts: fsts, err: err}
	}
}

func loadMGMsCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		mgms, err := client.MGMs(context.Background())
		return mgmsLoadedMsg{mgms: mgms, err: err}
	}
}

func loadEOSVersionCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		version, _ := client.EOSVersion(context.Background())
		return eosVersionLoadedMsg{version: version}
	}
}

func loadFileSystemsCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		fileSystems, err := client.FileSystems(context.Background())
		return fileSystemsLoadedMsg{fs: fileSystems, err: err}
	}
}

func loadSpacesCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		spaces, err := client.Spaces(context.Background())
		return spacesLoadedMsg{spaces: spaces, err: err}
	}
}

func loadGroupsCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		groups, err := client.Groups(context.Background())
		return groupsLoadedMsg{groups: groups, err: err}
	}
}

func loadNamespaceStatsCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		stats, err := client.NamespaceStats(context.Background())
		return namespaceStatsLoadedMsg{stats: stats, err: err}
	}
}

func loadDirectoryCmd(client *eos.Client, dirPath string) tea.Cmd {
	return func() tea.Msg {
		directory, err := client.ListPath(context.Background(), dirPath)
		return directoryLoadedMsg{directory: directory, err: err}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func newStyles() styles {
	return styles{
		app: lipgloss.NewStyle().
			Padding(0, 1),
		header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")),
		tab: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("245")),
		tabActive: lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")),
		panel: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1),
		panelDim: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1),
		selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")),
		label: lipgloss.NewStyle().
			Foreground(lipgloss.Color("109")),
		value: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("203")).
			Bold(true),
		status: lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")),
	}
}

func cleanPath(rawPath string) string {
	if rawPath == "" {
		return "/"
	}

	if !strings.HasPrefix(rawPath, "/") {
		return "/" + rawPath
	}

	return path.Clean(rawPath)
}

func parentPath(current string) string {
	if current == "/" {
		return "/"
	}

	return path.Dir(current)
}

func fallback(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}

	return value
}

func humanBytes(value uint64) string {
	if value < 1024 {
		return fmt.Sprintf("%d B", value)
	}

	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}
	size := float64(value)
	unit := -1
	for size >= 1024 && unit < len(units)-1 {
		size /= 1024
		unit++
	}

	if unit < 0 {
		return fmt.Sprintf("%d B", value)
	}

	return fmt.Sprintf("%.1f %s", size, units[unit])
}

func formatDuration(value time.Duration) string {
	if value <= 0 {
		return "-"
	}

	return value.Round(time.Second).String()
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}

	return value.Local().Format(time.RFC3339)
}

func formatTimeShort(value time.Time) string {
	if value.IsZero() {
		return "-"
	}

	return value.Local().Format("2006-01-02 15:04")
}

func truncate(value string, width int) string {
	// Ensure we don't have newlines which would break the 1-line layout.
	value = strings.ReplaceAll(value, "\n", " ")

	if width <= 0 || lipgloss.Width(value) <= width {
		return value
	}
	if width <= 1 {
		return "…"
	}

	runes := []rune(value)
	if len(runes) >= width {
		return string(runes[:width-1]) + "…"
	}

	return value
}

func padRight(value string, width int) string {
	value = truncate(value, width)
	return value + strings.Repeat(" ", max(0, width-lipgloss.Width(value)))
}

func padLeft(value string, width int) string {
	value = truncate(value, width)
	return strings.Repeat(" ", max(0, width-lipgloss.Width(value))) + value
}

func formatTableRow(columns []tableColumn, values []string) string {
	parts := make([]string, len(columns))
	for i, column := range columns {
		value := ""
		if i < len(values) {
			value = values[i]
		}
		if column.right {
			parts[i] = padLeft(value, column.min)
		} else {
			parts[i] = padRight(value, column.min)
		}
	}
	return strings.Join(parts, " ")
}

func allocateTableColumns(width int, columns []tableColumn) []tableColumn {
	if len(columns) == 0 {
		return nil
	}

	available := max(len(columns), width-(len(columns)-1))
	allocated := make([]tableColumn, len(columns))
	copy(allocated, columns)

	totalMin := 0
	totalWeight := 0
	for i := range allocated {
		allocated[i].min = max(1, max(allocated[i].min, lipgloss.Width(allocated[i].title)))
		if allocated[i].maxw > 0 {
			allocated[i].min = min(allocated[i].min, allocated[i].maxw)
		}
		totalMin += allocated[i].min
		// Columns already at their max don't participate in extra-space distribution.
		if allocated[i].maxw > 0 && allocated[i].min >= allocated[i].maxw {
			allocated[i].weight = 0
		}
		totalWeight += max(allocated[i].weight, 0)
	}

	if totalMin > available {
		overflow := totalMin - available
		for overflow > 0 {
			changed := false
			for i := range allocated {
				if overflow == 0 {
					break
				}
				if allocated[i].min > 1 {
					allocated[i].min--
					overflow--
					changed = true
				}
			}
			if !changed {
				break
			}
		}
		return allocated
	}

	extra := available - totalMin
	if totalWeight == 0 {
		totalWeight = len(allocated)
		for i := range allocated {
			if allocated[i].maxw == 0 || allocated[i].min < allocated[i].maxw {
				allocated[i].weight = 1
			}
		}
	}

	for i := range allocated {
		if extra == 0 {
			break
		}
		share := (extra * max(allocated[i].weight, 0)) / totalWeight
		if allocated[i].maxw > 0 {
			share = min(share, max(0, allocated[i].maxw-allocated[i].min))
		}
		allocated[i].min += share
		extra -= share
		totalWeight -= max(allocated[i].weight, 0)
	}

	for i := range allocated {
		if extra == 0 {
			break
		}
		if allocated[i].maxw == 0 || allocated[i].min < allocated[i].maxw {
			allocated[i].min++
			extra--
		}
	}

	return allocated
}

func visibleWindow(total, selected, capacity int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if capacity <= 0 || total <= capacity {
		return 0, total
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= total {
		selected = total - 1
	}

	start := selected - capacity/2
	if start < 0 {
		start = 0
	}
	maxStart := total - capacity
	if start > maxStart {
		start = maxStart
	}
	return start, min(total, start+capacity)
}

func renderScrollSummary(start, end, total int) string {
	if total <= 0 || end-start >= total {
		return ""
	}

	return fmt.Sprintf("  [%d-%d/%d]", start+1, end, total)
}

func panelContentWidth(width int) int {
	return max(1, width-4)
}

func panelContentHeight(height int) int {
	return max(1, height-2)
}

func fitLines(lines []string, height int) string {
	if height <= 0 {
		return ""
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func splitViewHeights(total int) (int, int) {
	if total <= 3 {
		return max(1, total-1), 1
	}

	available := max(2, total-1)
	listHeight := max(1, (available*2)/3)
	detailHeight := max(1, available-listHeight)

	if detailHeight < 4 {
		shift := min(listHeight-1, 4-detailHeight)
		listHeight -= shift
		detailHeight += shift
	}
	if listHeight < 4 {
		shift := min(detailHeight-1, 4-listHeight)
		detailHeight -= shift
		listHeight += shift
	}

	if listHeight+detailHeight > available {
		overflow := listHeight + detailHeight - available
		if listHeight >= detailHeight {
			listHeight = max(1, listHeight-overflow)
		} else {
			detailHeight = max(1, detailHeight-overflow)
		}
	}

	return listHeight, detailHeight
}

// adaptiveSplitHeights is like splitViewHeights but shrinks the list panel when
// its content is smaller than the default 2/3 allocation, giving the surplus to
// the detail panel.  naturalListContent is the number of content lines the list
// actually needs (excluding the 2-line border).
//
// The two panel borders together account for 4 lines (2 each), but because the
// body must fill height+2 rendered lines (to offset the -2 in View's bodyHeight
// formula), the net target is height+2.
func adaptiveSplitHeights(height, naturalListContent, naturalDetailContent int) (int, int) {
	target := height + 2
	naturalList := naturalListContent + 2
	naturalDetail := naturalDetailContent + 2

	// Constants for minimum usable heights.
	const minList = 6

	// Case 1: Everything fits. Expand the list to fill the target height
	// so the details box stays at its natural size at the bottom.
	if naturalList+naturalDetail <= target {
		return target - naturalDetail, naturalDetail
	}

	// Case 2: Space is tight. Prioritize the details box (bottom) natural height.
	// The list (top) should be shortened first.
	detailHeight := naturalDetail
	if target-detailHeight < minList {
		detailHeight = target - minList
	}

	listHeight := target - detailHeight

	// Final clamp to ensure nothing is below the absolute minimum of 4.
	listH := max(4, listHeight)
	return listH, target - listH
}

// contentAwareColumns adjusts the min width of each column to be at least as
// wide as the widest value in rows (or the column title, whichever is larger).
// This prevents short-content columns from consuming extra space via weight.
func contentAwareColumns(columns []tableColumn, rows [][]string) []tableColumn {
	result := make([]tableColumn, len(columns))
	copy(result, columns)
	for i := range result {
		w := lipgloss.Width(result[i].title)
		for _, row := range rows {
			if i < len(row) {
				if cw := lipgloss.Width(row[i]); cw > w {
					w = cw
				}
			}
		}
		if result[i].maxw > 0 && w > result[i].maxw {
			w = result[i].maxw
		}
		if w > result[i].min {
			result[i].min = w
		}
	}
	return result
}

func entryTypeLabel(entry eos.Entry) string {
	if entry.Kind == eos.EntryKindContainer {
		return "DIR"
	}

	return "FILE"
}

func entrySize(entry eos.Entry) string {
	if entry.Kind == eos.EntryKindContainer {
		return "-"
	}

	return humanBytes(entry.Size)
}

func usagePercent(used, capacity uint64) float64 {
	if capacity == 0 {
		return 0
	}

	return (float64(used) / float64(capacity)) * 100
}

func equivalentNodeSortValue(column int, a, b eos.FstRecord) bool {
	switch fstSortColumn(column) {
	case fstSortType:
		return a.Type == b.Type
	case fstSortHost:
		return a.Host == b.Host
	case fstSortStatus:
		return a.Status == b.Status
	case fstSortGeotag:
		return a.Geotag == b.Geotag
	case fstSortActivated:
		return a.Activated == b.Activated
	case fstSortNoFS:
		return a.FileSystemCount == b.FileSystemCount
	case fstSortHeartbeat:
		return a.HeartbeatDelta == b.HeartbeatDelta
	case fstSortEOSVersion:
		return a.EOSVersion == b.EOSVersion
	default:
		return a.Host == b.Host
	}
}

func equivalentFileSystemSortValue(column int, a, b eos.FileSystemRecord) bool {
	switch fsSortColumn(column) {
	case fsSortHost:
		return a.Host == b.Host
	case fsSortPort:
		return a.Port == b.Port
	case fsSortID:
		return a.ID == b.ID
	case fsSortPath:
		return a.Path == b.Path
	case fsSortGroup:
		return a.SchedGroup == b.SchedGroup
	case fsSortGeotag:
		return a.Geotag == b.Geotag
	case fsSortBoot:
		return a.Boot == b.Boot
	case fsSortConfigStatus:
		return a.ConfigStatus == b.ConfigStatus
	case fsSortDrain:
		return a.DrainStatus == b.DrainStatus
	case fsSortUsed:
		return usagePercent(a.UsedBytes, a.CapacityBytes) == usagePercent(b.UsedBytes, b.CapacityBytes)
	case fsSortStatus:
		return a.Active == b.Active
	case fsSortHealth:
		return a.Health == b.Health
	default:
		return a.ID == b.ID
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func clampIndex(index, length int) int {
	if length <= 0 {
		return 0
	}
	if index < 0 {
		return 0
	}
	if index >= length {
		return length - 1
	}
	return index
}

func max64(a int64, b int64) int64 {
	if a > b {
		return a
	}

	return b
}

func loadSpaceStatusCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		records, err := client.SpaceStatus(context.Background(), "default")
		return spaceStatusLoadedMsg{records: records, err: err}
	}
}

func loadIOShapingCmd(client *eos.Client, mode eos.IOShapingMode) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		records, err := client.IOShaping(ctx, mode)
		return ioShapingLoadedMsg{records: records, mode: mode, err: err}
	}
}

func ioShapingTickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return ioShapingTickMsg{}
	})
}

func loadIOShapingPoliciesCmd(client *eos.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		records, err := client.IOShapingPolicies(ctx)
		return ioShapingPoliciesLoadedMsg{records: records, err: err}
	}
}

func ioShapingPolicyTickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return ioShapingPolicyTickMsg{}
	})
}

func runSpaceConfigCmd(client *eos.Client, key, value string) tea.Cmd {
	return func() tea.Msg {
		err := client.SpaceConfig(context.Background(), "default", key, value)
		return spaceConfigResultMsg{err: err}
	}
}

// ---- Log overlay ------------------------------------------------------------

// selectedHostForView returns the hostname of the currently selected row in the
// active view, or empty string when no row is selected or the view has no host.
func (m model) selectedHostForView() string {
	switch m.activeView {
	case viewMGM:
		if m.mgmSelected >= 0 && m.mgmSelected < len(m.mgms) {
			return m.mgms[m.mgmSelected].Host
		}
	case viewQDB:
		if m.qdbSelected >= 0 && m.qdbSelected < len(m.mgms) {
			return m.mgms[m.qdbSelected].QDBHost
		}
	case viewFST:
		fsts := m.visibleFSTs()
		if m.fstSelected >= 0 && m.fstSelected < len(fsts) {
			return fsts[m.fstSelected].Host
		}
	case viewFileSystems:
		fss := m.visibleFileSystems()
		if m.fsSelected >= 0 && m.fsSelected < len(fss) {
			return fss[m.fsSelected].Host
		}
	}
	return ""
}

// logFileForView returns the log file path and a human-readable title for the
// currently active view.
func (m model) logFileForView() (filePath, title string) {
	switch m.activeView {
	case viewQDB:
		return "/var/log/eos/quarkdb/xrdlog.quarkdb", "QDB Log"
	case viewFST, viewFileSystems:
		return "/var/log/eos/fst/xrdlog.fst", "FST Log"
	default:
		return "/var/log/eos/mgm/xrdlog.mgm", "MGM Log"
	}
}

func (m model) openLogOverlay() (tea.Model, tea.Cmd) {
	if m.client == nil {
		return m, nil
	}
	filePath, title := m.logFileForView()
	host := m.selectedHostForView()

	logInput := textinput.New()
	logInput.Prompt = "grep> "
	logInput.CharLimit = 256

	vp := viewport.New(m.contentWidth()-4, max(4, m.height-10))
	vp.SetContent("Loading...")

	titleWithHost := title
	if host != "" {
		titleWithHost = fmt.Sprintf("%s  [%s]", title, host)
	}

	m.log = logOverlay{
		active:   true,
		host:     host,
		filePath: filePath,
		title:    titleWithHost,
		loading:  true,
		vp:       vp,
		input:    logInput,
	}
	return m, loadLogCmd(m.client, host, filePath)
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

func (m model) renderLogOverlay(height int) string {
	width := m.contentWidth()
	vpWidth := width - 4 // panel border + padding

	// Keep viewport sized to available space.
	filterHeight := 0
	if m.log.filtering {
		filterHeight = 2
	}
	vpHeight := max(4, height-4-filterHeight) // title bar (2) + border (2)
	m.log.vp.Width = vpWidth
	m.log.vp.Height = vpHeight

	// Title bar.
	filterInfo := ""
	if m.log.filter != "" {
		filterInfo = fmt.Sprintf("  [grep: %q  %d lines]", m.log.filter, len(m.log.filtered))
	}
	totalInfo := fmt.Sprintf("  %d lines", len(m.log.allLines))
	if m.log.loading {
		totalInfo = "  loading..."
	} else if m.log.err != nil {
		totalInfo = "  " + m.log.err.Error()
	}
	titleLine := m.styles.header.Render(m.log.title) +
		m.styles.label.Render("  "+m.log.filePath) +
		m.styles.value.Render(totalInfo+filterInfo)

	lines := []string{titleLine, ""}

	if m.log.err != nil && !m.log.loading {
		lines = append(lines, m.styles.error.Render(m.log.err.Error()))
	} else {
		lines = append(lines, m.log.vp.View())
	}

	if m.log.filtering {
		lines = append(lines, "", m.log.input.View())
	}

	inner := strings.Join(lines, "\n")
	return m.styles.panel.Width(width).Render(inner)
}

// applyLogFilter returns lines that case-insensitively contain filter.
// Empty filter returns all lines.
func applyLogFilter(lines []string, filter string) []string {
	if filter == "" {
		return lines
	}
	lower := strings.ToLower(filter)
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if strings.Contains(strings.ToLower(l), lower) {
			out = append(out, l)
		}
	}
	return out
}

func loadLogCmd(client *eos.Client, host, filePath string) tea.Cmd {
	return func() tea.Msg {
		out, err := client.TailLogOnHost(context.Background(), host, filePath, 2000)
		if err != nil {
			return logLoadedMsg{filePath: filePath, err: err}
		}
		raw := strings.TrimRight(string(out), "\n")
		lines := strings.Split(raw, "\n")
		return logLoadedMsg{filePath: filePath, lines: lines}
	}
}

// ---- Shell -----------------------------------------------------------------

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

func (m model) groupFilterColumnLabel() string {
	switch groupFilterColumn(m.groupFilter.column) {
	case groupFilterName:
		return "name"
	case groupFilterStatus:
		return "status"
	case groupFilterNoFS:
		return "nofs"
	case groupFilterCapacity:
		return "capacity"
	case groupFilterUsed:
		return "used"
	case groupFilterFree:
		return "free"
	case groupFilterFiles:
		return "files"
	default:
		return "name"
	}
}

func (m model) groupSortColumnLabel() string {
	switch groupSortColumn(m.groupSort.column) {
	case groupSortName:
		return "name"
	case groupSortStatus:
		return "status"
	case groupSortNoFS:
		return "nofs"
	case groupSortCapacity:
		return "capacity"
	case groupSortUsed:
		return "used"
	case groupSortFree:
		return "free"
	case groupSortFiles:
		return "files"
	case groupSortNone:
		return "none"
	default:
		return "name"
	}
}

func (m model) visibleGroups() []eos.GroupRecord {
	groups := append([]eos.GroupRecord(nil), m.groups...)
	if len(m.groupFilter.filters) > 0 {
		filtered := make([]eos.GroupRecord, 0, len(groups))
		for _, g := range groups {
			if m.matchesGroupFilters(g) {
				filtered = append(filtered, g)
			}
		}
		groups = filtered
	}
	if m.groupSort.column >= 0 {
		sort.SliceStable(groups, func(i, j int) bool {
			return m.lessGroup(groups[i], groups[j])
		})
	}
	return groups
}

func (m model) matchesGroupFilters(g eos.GroupRecord) bool {
	for col, filter := range m.groupFilter.filters {
		val := strings.ToLower(m.groupFilterValueForColumn(g, col))
		if !strings.Contains(val, strings.ToLower(filter)) {
			return false
		}
	}
	return true
}

func (m model) groupFilterValue(g eos.GroupRecord) string {
	return m.groupFilterValueForColumn(g, m.groupFilter.column)
}

func (m model) groupFilterValueForColumn(g eos.GroupRecord, column int) string {
	switch groupFilterColumn(column) {
	case groupFilterName:
		return g.Name
	case groupFilterStatus:
		return g.Status
	case groupFilterNoFS:
		return fmt.Sprintf("%d", g.NoFS)
	case groupFilterCapacity:
		return humanBytes(g.CapacityBytes)
	case groupFilterUsed:
		return humanBytes(g.UsedBytes)
	case groupFilterFree:
		return humanBytes(g.FreeBytes)
	case groupFilterFiles:
		return fmt.Sprintf("%d", g.NumFiles)
	default:
		return g.Name
	}
}

func (m model) lessGroup(a, b eos.GroupRecord) bool {
	less := false
	switch groupSortColumn(m.groupSort.column) {
	case groupSortName:
		less = a.Name < b.Name
	case groupSortStatus:
		less = a.Status < b.Status
	case groupSortNoFS:
		less = a.NoFS < b.NoFS
	case groupSortCapacity:
		less = a.CapacityBytes < b.CapacityBytes
	case groupSortUsed:
		less = a.UsedBytes < b.UsedBytes
	case groupSortFree:
		less = a.FreeBytes < b.FreeBytes
	case groupSortFiles:
		less = a.NumFiles < b.NumFiles
	}
	if m.groupSort.desc {
		return !less
	}
	return less
}

func groupColumnCount() int {
	return 7
}

func (m model) uniqueGroupValues(column int) []string {
	seen := make(map[string]bool)
	var values []string
	for _, g := range m.groups {
		val := m.groupFilterValueForColumn(g, column)
		if val != "" && !seen[val] {
			seen[val] = true
			values = append(values, val)
		}
	}
	sort.Strings(values)
	return values
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

func (m model) renderGroupsView(height int) string {
	const groupDetailLines = 6
	listHeight := max(4, height-groupDetailLines)
	detailHeight := groupDetailLines

	list := m.renderGroupsList(m.width, listHeight)
	details := m.renderGroupDetails(m.width, detailHeight)

	return lipgloss.JoinVertical(lipgloss.Left, list, details)
}

func (m model) renderGroupsList(width, height int) string {
	contentWidth := panelContentWidth(width)

	groups := m.visibleGroups()
	dataRows := make([][]string, len(groups))
	for i, g := range groups {
		dataRows[i] = []string{
			g.Name,
			g.Status,
			fmt.Sprintf("%d", g.NoFS),
			humanBytes(g.CapacityBytes),
			humanBytes(g.UsedBytes),
			humanBytes(g.FreeBytes),
			fmt.Sprintf("%d", g.NumFiles),
		}
	}

	columnDefs := contentAwareColumns([]tableColumn{
		{title: "name", min: 10, weight: 3},
		{title: "status", min: 6, weight: 1},
		{title: "nofs", min: 4, weight: 0, right: true},
		{title: "capacity", min: 8, weight: 0, right: true},
		{title: "used", min: 8, weight: 0, right: true},
		{title: "free", min: 8, weight: 0, right: true},
		{title: "files", min: 5, weight: 0, right: true},
	}, dataRows)

	columns := allocateTableColumns(contentWidth, columnDefs)

	title := m.styles.label.Render("EOS Groups")
	lines := []string{
		title,
		"",
		m.renderGroupHeaderRow(columns),
	}

	if m.groupsLoading {
		lines = append(lines, "Loading groups...")
	} else if m.groupsErr != nil {
		lines = append(lines, m.styles.error.Render(m.groupsErr.Error()))
	} else if len(groups) == 0 {
		lines = append(lines, "(no groups)")
	} else {
		start, end := visibleWindow(len(groups), m.groupsSelected, max(1, panelContentHeight(height)-len(lines)))
		lines[0] = title + renderScrollSummary(start, end, len(groups))
		for i := start; i < end; i++ {
			g := groups[i]
			row := []string{
				g.Name,
				g.Status,
				fmt.Sprintf("%d", g.NoFS),
				humanBytes(g.CapacityBytes),
				humanBytes(g.UsedBytes),
				humanBytes(g.FreeBytes),
				fmt.Sprintf("%d", g.NumFiles),
			}
			line := formatTableRow(columns, row)
			if i == m.groupsSelected {
				line = m.styles.selected.Width(contentWidth).Render(line)
			}
			lines = append(lines, line)
		}
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, panelContentHeight(height)))
}

func (m model) renderGroupHeaderRow(columns []tableColumn) string {
	labels := []string{"name", "status", "nofs", "capacity", "used", "free", "files"}
	return m.renderSelectableHeaderRow(columns, labels, m.groupsColumnSelected, m.groupSort, m.groupFilter)
}

func (m model) renderGroupDetails(width, height int) string {
	contentWidth := panelContentWidth(width)
	groups := m.visibleGroups()
	if len(groups) == 0 || m.groupsSelected < 0 || m.groupsSelected >= len(groups) {
		return m.styles.panelDim.Width(contentWidth).Height(panelContentHeight(height)).Render("no group selected")
	}

	g := groups[m.groupsSelected]
	title := m.styles.label.Render("Selected Group") + " " + g.Name

	box := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		m.metricLine("Status", g.Status, "Filesystems", fmt.Sprintf("%d", g.NoFS)),
		m.metricLine("Capacity", humanBytes(g.CapacityBytes), "Used", humanBytes(g.UsedBytes)),
		m.metricLine("Free", humanBytes(g.FreeBytes), "Files", fmt.Sprintf("%d", g.NumFiles)),
	)

	return m.styles.panelDim.Width(contentWidth).Height(panelContentHeight(height)).Render(box)
}
