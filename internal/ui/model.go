package ui

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/lobis/eos-tui/internal/eosgrpc"
)

const refreshInterval = 5 * time.Second

type viewID int

const (
	viewNodes viewID = iota
	viewFileSystems
	viewNamespace
	viewSpaces
	viewNamespaceStats
)

type infraLoadedMsg struct {
	stats eosgrpc.NodeStats
	nodes []eosgrpc.NodeRecord
	fs    []eosgrpc.FileSystemRecord
	err   error
}

type nodeStatsLoadedMsg struct {
	stats eosgrpc.NodeStats
	err   error
}

type nodesLoadedMsg struct {
	nodes []eosgrpc.NodeRecord
	err   error
}

type fileSystemsLoadedMsg struct {
	fs  []eosgrpc.FileSystemRecord
	err error
}

type spacesLoadedMsg struct {
	spaces []eosgrpc.SpaceRecord
	err    error
}

type namespaceStatsLoadedMsg struct {
	stats eosgrpc.NamespaceStats
	err   error
}

type directoryLoadedMsg struct {
	directory eosgrpc.Directory
	err       error
}

type tickMsg time.Time

type model struct {
	client   *eosgrpc.Client
	endpoint string

	width  int
	height int

	activeView viewID

	nodeStatsLoading   bool
	nodesLoading       bool
	fileSystemsLoading bool
	nodeStatsErr       error
	nodesErr           error
	fileSystemsErr     error
	nodeStats          eosgrpc.NodeStats
	nodes              []eosgrpc.NodeRecord
	nodeSelected       int
	nodeColumnSelected int
	fileSystems        []eosgrpc.FileSystemRecord
	fsSelected         int
	fsColumnSelected   int

	spaces               []eosgrpc.SpaceRecord
	spacesLoading        bool
	spacesErr            error
	spacesSelected       int
	spacesColumnSelected int

	namespaceStats eosgrpc.NamespaceStats
	nsStatsLoading bool
	nsStatsErr     error

	directory  eosgrpc.Directory
	nsLoaded   bool
	nsLoading  bool
	nsErr      error
	nsSelected int

	status string

	nodeFilter filterState
	nodeSort   sortState
	fsFilter   filterState
	fsSort     sortState
	popup      filterPopup

	styles styles
}

type filterPopup struct {
	active bool
	view   viewID
	column int
	input  textinput.Model
	table  table.Model
	values []string
}

type filterState struct {
	column  int
	filters map[int]string
}

type sortState struct {
	column int
	desc   bool
}

type nodeFilterColumn int
type nodeSortColumn int
type fsFilterColumn int
type fsSortColumn int

const (
	nodeFilterType nodeFilterColumn = iota
	nodeFilterHostPort
	nodeFilterGeotag
	nodeFilterStatus
	nodeFilterActivated
	nodeFilterHeartbeatDelta
	nodeFilterNoFS
)

const nodeSortNone nodeSortColumn = -1

const (
	nodeSortType nodeSortColumn = iota
	nodeSortHostPort
	nodeSortGeotag
	nodeSortStatus
	nodeSortActivated
	nodeSortHeartbeat
	nodeSortNoFS
)

const nodeSortFileSystems = nodeSortNoFS

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
	weight int
	right  bool
}

func NewModel(client *eosgrpc.Client, endpoint, rootPath string) tea.Model {
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
		activeView:         viewNodes,
		nodeStatsLoading:   true,
		nodesLoading:       true,
		fileSystemsLoading: true,
		spacesLoading:      true,
		nsStatsLoading:     true,
		nsLoading:          false,
		directory: eosgrpc.Directory{
			Path: cleanPath(rootPath),
		},
		status:             "Loading EOS state...",
		nodeColumnSelected: int(nodeFilterHostPort),
		fsColumnSelected:   int(fsFilterHost),
		nodeSort:           sortState{column: int(nodeSortNone)},
		fsSort:             sortState{column: int(fsSortNone)},
		nodeFilter:         filterState{filters: map[int]string{}},
		fsFilter:           filterState{filters: map[int]string{}},
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
		if m.popup.active {
			return m.updatePopup(msg)
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.activeView = (m.activeView + 1) % 5
			if m.activeView == viewNamespace {
				return m.maybeLoadNamespace()
			}
		case "1":
			m.activeView = viewNodes
		case "2":
			m.activeView = viewFileSystems
		case "3":
			m.activeView = viewNamespace
			return m.maybeLoadNamespace()
		case "4":
			m.activeView = viewSpaces
		case "5":
			m.activeView = viewNamespaceStats
		case "r":
			return m.refreshActiveView()
		}

		switch m.activeView {
		case viewNodes:
			return m.updateNodesKeys(msg)
		case viewFileSystems:
			return m.updateFileSystemKeys(msg)
		case viewNamespace:
			return m.updateNamespaceKeys(msg)
		case viewSpaces:
			return m.updateSpacesKeys(msg)
		case viewNamespaceStats:
			// namespace stats view is read-only, just refresh on 'r'
		}
	case infraLoadedMsg:
		m.nodeStatsLoading = false
		m.nodesLoading = false
		m.fileSystemsLoading = false
		if msg.err != nil {
			m.nodeStatsErr = msg.err
			m.nodesErr = msg.err
			m.fileSystemsErr = msg.err
			m.status = fmt.Sprintf("Infrastructure refresh failed: %v", msg.err)
		} else {
			m.nodeStats = msg.stats
			m.nodes = msg.nodes
			m.fileSystems = msg.fs
			m.nodeSelected = clampIndex(m.nodeSelected, len(m.visibleNodes()))
			m.fsSelected = clampIndex(m.fsSelected, len(m.visibleFileSystems()))
			m.status = fmt.Sprintf("Connected to %s", m.endpoint)
		}
	case nodeStatsLoadedMsg:
		m.nodeStatsLoading = false
		m.nodeStatsErr = msg.err
		if msg.err != nil {
			m.status = fmt.Sprintf("Cluster summary refresh failed: %v", msg.err)
		} else {
			m.nodeStats = msg.stats
			m.status = fmt.Sprintf("Connected to %s", m.endpoint)
		}
	case nodesLoadedMsg:
		m.nodesLoading = false
		m.nodesErr = msg.err
		if msg.err != nil {
			m.status = fmt.Sprintf("Node list refresh failed: %v", msg.err)
		} else {
			m.nodes = msg.nodes
			m.nodeSelected = clampIndex(m.nodeSelected, len(m.visibleNodes()))
			m.status = fmt.Sprintf("Connected to %s", m.endpoint)
		}
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
	case spacesLoadedMsg:
		m.spacesLoading = false
		m.spacesErr = msg.err
		if msg.err != nil {
			m.status = fmt.Sprintf("Spaces refresh failed: %v", msg.err)
		} else {
			m.spaces = msg.spaces
			m.spacesSelected = clampIndex(m.spacesSelected, len(m.spaces))
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
	case tickMsg:
		return m, tea.Batch(tickCmd(), loadInfraCmd(m.client))
	}

	return m, nil
}

func (m model) View() string {
	header := m.renderHeader()
	footer := m.renderFooter()
	bodyHeight := max(4, m.height-lipgloss.Height(header)-lipgloss.Height(footer)-2)
	body := m.renderBody(bodyHeight)
	if m.popup.active {
		body = m.renderBodyWithPopup(body, bodyHeight)
	}

	return m.styles.app.
		Render(lipgloss.JoinVertical(lipgloss.Left, header, body, footer))
}

func (m model) renderBodyWithPopup(body string, height int) string {
	popup := m.renderFilterPopup()
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
	case viewNamespaceStats:
		m.nsStatsLoading = true
		m.nsStatsErr = nil
		m.status = "Refreshing namespace stats..."
		return m, loadNamespaceStatsCmd(m.client)
	default:
		m.nodeStatsLoading = true
		m.nodesLoading = true
		m.fileSystemsLoading = true
		m.nodeStatsErr = nil
		m.nodesErr = nil
		m.fileSystemsErr = nil
		m.status = "Refreshing node and filesystem state..."
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

func (m model) updateNodesKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	nodes := m.visibleNodes()
	switch msg.String() {
	case "f":
		m.nodeFilter.column = m.nodeColumnSelected
		m.openFilterPopup()
		return m, nil
	case "/":
		m.nodeFilter.column = m.nodeColumnSelected
		m.openFilterPopup()
		return m, nil
	case "left":
		m.nodeColumnSelected = max(0, m.nodeColumnSelected-1)
		m.status = fmt.Sprintf("Selected node column: %s", m.nodeSelectedColumnLabel())
	case "right":
		m.nodeColumnSelected = min(nodeColumnCount()-1, m.nodeColumnSelected+1)
		m.status = fmt.Sprintf("Selected node column: %s", m.nodeSelectedColumnLabel())
	case "s", "enter":
		m.nodeSort = m.nextNodeSortState()
		m.nodeSelected = clampIndex(0, len(m.visibleNodes()))
		m.status = fmt.Sprintf("Node sort: %s", m.nodeSortStateLabel())
	case "c":
		delete(m.nodeFilter.filters, m.nodeColumnSelected)
		m.nodeFilter.column = m.nodeColumnSelected
		m.status = fmt.Sprintf("Cleared node filter on %s", m.nodeSelectedColumnLabel())
	case "up", "k":
		if m.nodeSelected > 0 {
			m.nodeSelected--
		}
	case "down", "j":
		if m.nodeSelected < len(nodes)-1 {
			m.nodeSelected++
		}
	case "g":
		m.nodeSelected = 0
	case "G":
		m.nodeSelected = max(0, len(nodes)-1)
	}

	return m, nil
}

func (m model) updateFileSystemKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	fileSystems := m.visibleFileSystems()
	switch msg.String() {
	case "f":
		m.fsFilter.column = m.fsColumnSelected
		m.openFilterPopup()
		return m, nil
	case "/":
		m.fsFilter.column = m.fsColumnSelected
		m.openFilterPopup()
		return m, nil
	case "left":
		m.fsColumnSelected = max(0, m.fsColumnSelected-1)
		m.status = fmt.Sprintf("Selected filesystem column: %s", m.fsSelectedColumnLabel())
	case "right":
		m.fsColumnSelected = min(fsColumnCount()-1, m.fsColumnSelected+1)
		m.status = fmt.Sprintf("Selected filesystem column: %s", m.fsSelectedColumnLabel())
	case "s", "enter":
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
	case "g":
		m.fsSelected = 0
	case "G":
		m.fsSelected = max(0, len(fileSystems)-1)
	}

	return m, nil
}

func (m model) updateNamespaceKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.nsSelected > 0 {
			m.nsSelected--
		}
	case "down", "j":
		if m.nsSelected < len(m.directory.Entries)-1 {
			m.nsSelected++
		}
	case "g":
		m.nsSelected = 0
		m.nsLoading = true
		m.status = "Jumping to / ..."
		return m, loadDirectoryCmd(m.client, "/")
	case "backspace", "h", "left":
		parent := parentPath(m.directory.Path)
		if parent != m.directory.Path {
			m.nsSelected = 0
			m.nsLoading = true
			m.status = fmt.Sprintf("Opening %s...", parent)
			return m, loadDirectoryCmd(m.client, parent)
		}
	case "enter", "l", "right":
		entry, ok := m.selectedNamespaceEntry()
		if ok && entry.Kind == eosgrpc.EntryKindContainer {
			m.nsSelected = 0
			m.nsLoading = true
			m.status = fmt.Sprintf("Opening %s...", entry.Path)
			return m, loadDirectoryCmd(m.client, entry.Path)
		}
	}

	return m, nil
}

func (m model) updateSpacesKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.spacesSelected > 0 {
			m.spacesSelected--
		}
	case "down", "j":
		if m.spacesSelected < len(m.spaces)-1 {
			m.spacesSelected++
		}
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

func (m model) selectedNode() (eosgrpc.NodeRecord, bool) {
	nodes := m.visibleNodes()
	if len(nodes) == 0 || m.nodeSelected < 0 || m.nodeSelected >= len(nodes) {
		return eosgrpc.NodeRecord{}, false
	}

	return nodes[m.nodeSelected], true
}

func (m model) selectedFileSystem() (eosgrpc.FileSystemRecord, bool) {
	fileSystems := m.visibleFileSystems()
	if len(fileSystems) == 0 || m.fsSelected < 0 || m.fsSelected >= len(fileSystems) {
		return eosgrpc.FileSystemRecord{}, false
	}

	return fileSystems[m.fsSelected], true
}

func (m model) selectedNamespaceEntry() (eosgrpc.Entry, bool) {
	if len(m.directory.Entries) == 0 || m.nsSelected < 0 || m.nsSelected >= len(m.directory.Entries) {
		return eosgrpc.Entry{}, false
	}

	return m.directory.Entries[m.nsSelected], true
}

func (m model) renderHeader() string {
	tabNodes := m.styles.tab.Render("1 Nodes")
	tabFS := m.styles.tab.Render("2 Filesystems")
	tabNS := m.styles.tab.Render("3 Namespace")
	tabSpaces := m.styles.tab.Render("4 Spaces")
	tabNSStats := m.styles.tab.Render("5 NS Stats")

	switch m.activeView {
	case viewNodes:
		tabNodes = m.styles.tabActive.Render("1 Nodes")
	case viewFileSystems:
		tabFS = m.styles.tabActive.Render("2 Filesystems")
	case viewNamespace:
		tabNS = m.styles.tabActive.Render("3 Namespace")
	case viewSpaces:
		tabSpaces = m.styles.tabActive.Render("4 Spaces")
	case viewNamespaceStats:
		tabNSStats = m.styles.tabActive.Render("5 NS Stats")
	}

	left := lipgloss.JoinHorizontal(
		lipgloss.Left,
		m.styles.header.Render("EOS TUI"),
		"  ",
		tabNodes,
		" ",
		tabFS,
		" ",
		tabNS,
		" ",
		tabSpaces,
		" ",
		tabNSStats,
	)
	right := m.styles.label.Render("target ") + m.styles.value.Render(m.endpoint)
	spacerWidth := max(1, m.contentWidth()-lipgloss.Width(left)-lipgloss.Width(right))

	return lipgloss.JoinHorizontal(lipgloss.Left, left, strings.Repeat(" ", spacerWidth), right)
}

func (m model) renderBody(availableHeight int) string {
	switch m.activeView {
	case viewFileSystems:
		return m.renderFileSystemsView(availableHeight)
	case viewNamespace:
		return m.renderNamespaceView(availableHeight)
	case viewSpaces:
		return m.renderSpacesView(availableHeight)
	case viewNamespaceStats:
		return m.renderNamespaceStatsView(availableHeight)
	default:
		return m.renderNodesView(availableHeight)
	}
}

func (m model) renderNodesView(height int) string {
	listHeight, detailHeight := splitViewHeights(height)
	width := m.contentWidth()

	list := m.renderNodesList(width, listHeight)
	details := m.renderNodeDetails(width, detailHeight)
	return lipgloss.JoinVertical(lipgloss.Left, list, details)
}

func (m model) renderNodesList(width, height int) string {
	contentWidth := panelContentWidth(width)
	columns := allocateTableColumns(contentWidth, []tableColumn{
		{title: "type", min: 10, weight: 2},
		{title: "hostport", min: 28, weight: 6},
		{title: "geotag", min: 20, weight: 4},
		{title: "status", min: 8, weight: 1},
		{title: "activated", min: 10, weight: 1},
		{title: "heartbeatdelta", min: 16, weight: 1, right: true},
		{title: "nofs", min: 4, weight: 1, right: true},
	})

	title := m.styles.label.Render("Cluster Summary")
	nodes := m.visibleNodes()
	lines := []string{
		title + m.renderNodeControls(),
		m.metricLine("Health", fallback(m.nodeStats.State, "-"), "Threads", fmt.Sprintf("%d", m.nodeStats.ThreadCount)),
		m.metricLine("Files", fmt.Sprintf("%d", m.nodeStats.FileCount), "Dirs", fmt.Sprintf("%d", m.nodeStats.DirCount)),
		m.metricLine("Uptime", formatDuration(m.nodeStats.Uptime), "FDs", fmt.Sprintf("%d", m.nodeStats.FileDescs)),
		"",
		m.renderNodeHeaderRow(columns),
	}

	if m.nodeStatsLoading {
		lines[1] = m.styles.value.Render("Loading cluster summary...")
		lines[2] = ""
		lines[3] = ""
	}

	if m.nodesLoading {
		lines = append(lines, "Loading node list...")
	} else if m.nodesErr != nil {
		lines = append(lines, m.styles.error.Render(m.nodesErr.Error()))
	} else if len(nodes) == 0 {
		lines = append(lines, "(no nodes)")
	} else {
		start, end := visibleWindow(len(nodes), m.nodeSelected, max(1, panelContentHeight(height)-len(lines)))
		lines[0] = title + m.renderNodeControls() + renderScrollSummary(start, end, len(nodes))
		for i := start; i < end; i++ {
			node := nodes[i]
			line := formatTableRow(columns, []string{
				node.Type,
				node.HostPort,
				node.Geotag,
				node.Status,
				node.Activated,
				fmt.Sprintf("%d", node.HeartbeatDelta),
				fmt.Sprintf("%d", node.FileSystemCount),
			})
			if i == m.nodeSelected {
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
		truncate(node.HostPort, max(10, width-4)),
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
	listHeight, detailHeight := splitViewHeights(height)
	width := m.contentWidth()

	list := m.renderFileSystemsList(width, listHeight)
	details := m.renderFileSystemDetails(width, detailHeight)
	return lipgloss.JoinVertical(lipgloss.Left, list, details)
}

func (m model) renderFileSystemsList(width, height int) string {
	contentWidth := panelContentWidth(width)
	columns := allocateTableColumns(contentWidth, []tableColumn{
		{title: "host", min: 22, weight: 4},
		{title: "port", min: 5, weight: 1, right: true},
		{title: "id", min: 4, weight: 1, right: true},
		{title: "path", min: 18, weight: 4},
		{title: "schedgroup", min: 12, weight: 2},
		{title: "geotag", min: 12, weight: 2},
		{title: "boot", min: 8, weight: 1},
		{title: "configstatus", min: 12, weight: 1},
		{title: "drain", min: 8, weight: 1},
		{title: "usage %", min: 8, weight: 1, right: true},
		{title: "active", min: 8, weight: 1},
		{title: "health", min: 12, weight: 2},
	})

	title := m.styles.label.Render("EOS Filesystems")
	fileSystems := m.visibleFileSystems()
	lines := []string{
		title + m.renderFileSystemControls(),
		"",
		m.renderFileSystemHeaderRow(columns),
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
			fs := fileSystems[i]
			line := formatTableRow(columns, []string{
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
			})
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
	listHeight, detailHeight := splitViewHeights(height)
	width := m.contentWidth()

	list := m.renderSpacesList(width, listHeight)
	details := m.renderSpaceDetails(width, detailHeight)
	return lipgloss.JoinVertical(lipgloss.Left, list, details)
}

func (m model) renderSpacesList(width, height int) string {
	contentWidth := panelContentWidth(width)
	columns := allocateTableColumns(contentWidth, []tableColumn{
		{title: "name", min: 16, weight: 3},
		{title: "type", min: 10, weight: 2},
		{title: "status", min: 10, weight: 2},
		{title: "groups", min: 8, weight: 1, right: true},
		{title: "files", min: 10, weight: 1, right: true},
		{title: "dirs", min: 10, weight: 1, right: true},
		{title: "usage %", min: 8, weight: 1, right: true},
	})

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
			space := m.spaces[i]
			line := formatTableRow(columns, []string{
				space.Name,
				space.Type,
				space.Status,
				fmt.Sprintf("%d", space.Groups),
				fmt.Sprintf("%d", space.NumFiles),
				fmt.Sprintf("%d", space.NumContainers),
				fmt.Sprintf("%.2f", usagePercent(space.UsedBytes, space.CapacityBytes)),
			})
			if i == m.spacesSelected {
				line = m.styles.selected.Width(contentWidth).Render(line)
			}
			lines = append(lines, line)
		}
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, panelContentHeight(height)))
}

func (m model) renderSpaceHeaderRow(columns []tableColumn) string {
	labels := []string{"name", "type", "status", "groups", "files", "dirs", "usage %"}
	cells := make([]string, 0, len(columns))
	for i, column := range columns {
		label := ""
		if i < len(labels) {
			label = labels[i]
		}
		cell := padRight(label, column.min)
		cell = m.styles.label.Render(cell)
		cells = append(cells, cell)
	}
	return strings.Join(cells, " ")
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
	if m.nsStatsLoading {
		return m.styles.panelDim.Render("Loading namespace statistics...")
	}
	if m.nsStatsErr != nil {
		return m.styles.panelDim.Render(fmt.Sprintf("Error loading namespace stats: %v", m.nsStatsErr))
	}

	width := m.contentWidth()
	contentWidth := panelContentWidth(width)

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

func (m model) renderNamespaceView(height int) string {
	listHeight, detailHeight := splitViewHeights(height)
	width := m.contentWidth()

	list := m.renderNamespaceList(width, listHeight)
	details := m.renderNamespaceDetails(width, detailHeight)
	return lipgloss.JoinVertical(lipgloss.Left, list, details)
}

func (m model) renderNamespaceList(width, height int) string {
	contentWidth := panelContentWidth(width)
	columns := allocateTableColumns(contentWidth, []tableColumn{
		{title: "TYPE", min: 4, weight: 1},
		{title: "NAME", min: 24, weight: 6},
		{title: "SIZE", min: 10, weight: 2, right: true},
		{title: "UID", min: 6, weight: 1, right: true},
		{title: "GID", min: 6, weight: 1, right: true},
		{title: "MODIFIED", min: 16, weight: 2},
	})

	title := m.styles.label.Render("Namespace Path ") + m.styles.value.Render(m.directory.Path)
	lines := []string{
		title,
		"",
		formatTableRow(columns, []string{"TYPE", "NAME", "SIZE", "UID", "GID", "MODIFIED"}),
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

	if target.Kind == eosgrpc.EntryKindContainer {
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
	keys := "tab switch view • r refresh • j/k move • ←/→ column • s sort asc/desc/off • f filter • c clear filter • / search values • 1/2/3 jump • q quit"
	if m.activeView == viewNamespace {
		keys = "tab switch view • r refresh • j/k move • 1/2/3 jump • namespace: enter open, h/backspace up, g root • q quit"
	}
	lines := []string{keys, m.status}
	return m.styles.status.Width(m.contentWidth()).Render(strings.Join(lines, "\n"))
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

func (m model) visibleNodes() []eosgrpc.NodeRecord {
	nodes := append([]eosgrpc.NodeRecord(nil), m.nodes...)
	if len(m.nodeFilter.filters) > 0 {
		filtered := make([]eosgrpc.NodeRecord, 0, len(nodes))
		for _, node := range nodes {
			if m.matchesNodeFilters(node) {
				filtered = append(filtered, node)
			}
		}
		nodes = filtered
	}
	if m.nodeSort.column >= 0 {
		sort.SliceStable(nodes, func(i, j int) bool {
			return m.lessNode(nodes[i], nodes[j])
		})
	}
	return nodes
}

func (m model) visibleFileSystems() []eosgrpc.FileSystemRecord {
	fileSystems := append([]eosgrpc.FileSystemRecord(nil), m.fileSystems...)
	if len(m.fsFilter.filters) > 0 {
		filtered := make([]eosgrpc.FileSystemRecord, 0, len(fileSystems))
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

func (m model) nodeFilterValue(node eosgrpc.NodeRecord) string {
	return m.nodeFilterValueForColumn(node, m.nodeFilter.column)
}

func (m model) nodeFilterValueForColumn(node eosgrpc.NodeRecord, column int) string {
	switch nodeFilterColumn(column) {
	case nodeFilterType:
		return node.Type
	case nodeFilterStatus:
		return node.Status
	case nodeFilterGeotag:
		return node.Geotag
	case nodeFilterActivated:
		return node.Activated
	case nodeFilterHeartbeatDelta:
		return fmt.Sprintf("%d", node.HeartbeatDelta)
	case nodeFilterNoFS:
		return fmt.Sprintf("%d", node.FileSystemCount)
	default:
		return node.HostPort
	}
}

func (m model) fsFilterValue(fs eosgrpc.FileSystemRecord) string {
	return m.fsFilterValueForColumn(fs, m.fsFilter.column)
}

func (m model) fsFilterValueForColumn(fs eosgrpc.FileSystemRecord, column int) string {
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

func (m model) matchesNodeFilters(node eosgrpc.NodeRecord) bool {
	for column, query := range m.nodeFilter.filters {
		if query == "" {
			continue
		}
		value := strings.ToLower(m.nodeFilterValueForColumn(node, column))
		if !strings.Contains(value, strings.ToLower(query)) {
			return false
		}
	}
	return true
}

func (m model) matchesFileSystemFilters(fs eosgrpc.FileSystemRecord) bool {
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

func (m model) lessNode(a, b eosgrpc.NodeRecord) bool {
	var less bool
	switch nodeSortColumn(m.nodeSort.column) {
	case nodeSortType:
		less = strings.Compare(a.Type, b.Type) < 0
	case nodeSortHostPort:
		less = strings.Compare(a.HostPort, b.HostPort) < 0
	case nodeSortStatus:
		less = strings.Compare(a.Status, b.Status) < 0
	case nodeSortGeotag:
		less = strings.Compare(a.Geotag, b.Geotag) < 0
	case nodeSortActivated:
		less = strings.Compare(a.Activated, b.Activated) < 0
	case nodeSortNoFS:
		less = a.FileSystemCount < b.FileSystemCount
	case nodeSortHeartbeat:
		less = a.HeartbeatDelta < b.HeartbeatDelta
	default:
		less = strings.Compare(a.HostPort, b.HostPort) < 0
	}
	if equivalentNodeSortValue(m.nodeSort.column, a, b) {
		less = strings.Compare(a.HostPort, b.HostPort) < 0
	}
	if m.nodeSort.desc {
		return !less
	}
	return less
}

func (m model) lessFileSystem(a, b eosgrpc.FileSystemRecord) bool {
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
		m.nodeSelectedColumnLabel(),
		len(m.nodeFilter.filters),
		filterValueLabel(m.nodeFilter.filters[m.nodeColumnSelected], m.popup.active && m.popup.view == viewNodes, m.popup.input.Value()),
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

func (m model) renderNodeHeaderRow(columns []tableColumn) string {
	labels := []string{"type", "hostport", "geotag", "status", "activated", "heartbeatdelta", "nofs"}
	return m.renderSelectableHeaderRow(columns, labels, m.nodeColumnSelected, m.nodeSort, m.nodeFilter)
}

func (m model) renderFileSystemHeaderRow(columns []tableColumn) string {
	labels := []string{"host", "port", "id", "path", "schedgroup", "geotag", "boot", "configstatus", "drain", "usage %", "active", "health"}
	return m.renderSelectableHeaderRow(columns, labels, m.fsColumnSelected, m.fsSort, m.fsFilter)
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

func (m model) nodeFilterColumnLabel() string {
	switch nodeFilterColumn(m.nodeFilter.column) {
	case nodeFilterType:
		return "type"
	case nodeFilterStatus:
		return "status"
	case nodeFilterGeotag:
		return "geotag"
	case nodeFilterActivated:
		return "activated"
	case nodeFilterHeartbeatDelta:
		return "heartbeatdelta"
	case nodeFilterNoFS:
		return "nofs"
	default:
		return "hostport"
	}
}

func (m model) nodeSortColumnLabel() string {
	switch nodeSortColumn(m.nodeSort.column) {
	case nodeSortType:
		return "type"
	case nodeSortHostPort:
		return "hostport"
	case nodeSortStatus:
		return "status"
	case nodeSortGeotag:
		return "geotag"
	case nodeSortActivated:
		return "activated"
	case nodeSortNoFS:
		return "nofs"
	case nodeSortHeartbeat:
		return "heartbeatdelta"
	case nodeSortNone:
		return "none"
	default:
		return "hostport"
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
	default:
		return m.nodeFilterColumnLabel()
	}
}

func (m *model) openFilterPopup() {
	m.popup.active = true
	m.popup.view = m.activeView
	if m.activeView == viewFileSystems {
		m.popup.column = m.fsColumnSelected
		m.popup.input.SetValue(m.fsFilter.filters[m.fsColumnSelected])
	} else {
		m.popup.column = m.nodeColumnSelected
		m.popup.input.SetValue(m.nodeFilter.filters[m.nodeColumnSelected])
	}
	m.popup.input.CursorEnd()
	m.popup.input.Focus()
	m.popup.table.Focus()
	m.popup.table.SetCursor(0)
	m.updatePopupRows()
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
	default:
		m.nodeFilter.column = m.popup.column
		if value == "" {
			delete(m.nodeFilter.filters, m.popup.column)
		} else {
			m.nodeFilter.filters[m.popup.column] = value
		}
		m.nodeSelected = clampIndex(0, len(m.visibleNodes()))
		m.closeFilterPopup(fmt.Sprintf("Node filters active: %d", len(m.nodeFilter.filters)))
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
	switch m.popup.view {
	case viewFileSystems:
		for _, fs := range m.fileSystems {
			value := m.fsFilterValueForColumn(fs, m.popup.column)
			if !seen[value] {
				seen[value] = true
				values = append(values, value)
			}
		}
	default:
		for _, node := range m.nodes {
			value := m.nodeFilterValueForColumn(node, m.popup.column)
			if !seen[value] {
				seen[value] = true
				values = append(values, value)
			}
		}
	}
	sort.Strings(values[1:])
	return values
}

func nodeColumnCount() int {
	return 7
}

func fsColumnCount() int {
	return 12
}

func (m model) nodeSelectedColumnLabel() string {
	column := m.nodeFilter.column
	m.nodeFilter.column = m.nodeColumnSelected
	label := m.nodeFilterColumnLabel()
	m.nodeFilter.column = column
	return label
}

func (m model) fsSelectedColumnLabel() string {
	column := m.fsFilter.column
	m.fsFilter.column = m.fsColumnSelected
	label := m.fsFilterColumnLabel()
	m.fsFilter.column = column
	return label
}

func (m model) nodeSortStateLabel() string {
	if m.nodeSort.column < 0 {
		return "none"
	}
	return fmt.Sprintf("%s %s", m.nodeSortColumnLabel(), sortDirectionLabel(m.nodeSort.desc))
}

func (m model) fsSortStateLabel() string {
	if m.fsSort.column < 0 {
		return "none"
	}
	return fmt.Sprintf("%s %s", m.fsSortColumnLabel(), sortDirectionLabel(m.fsSort.desc))
}

func (m model) nextNodeSortState() sortState {
	selected := m.nodeColumnSelected
	if m.nodeSort.column != selected {
		return sortState{column: selected}
	}
	if !m.nodeSort.desc {
		return sortState{column: selected, desc: true}
	}
	return sortState{column: int(nodeSortNone)}
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
	switch nodeFilterColumn(column) {
	case nodeFilterType, nodeFilterStatus, nodeFilterActivated:
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

func loadInfraCmd(client *eosgrpc.Client) tea.Cmd {
	return tea.Batch(
		loadNodeStatsCmd(client),
		loadNodesCmd(client),
		loadFileSystemsCmd(client),
		loadSpacesCmd(client),
		loadNamespaceStatsCmd(client),
	)
}

func loadNodeStatsCmd(client *eosgrpc.Client) tea.Cmd {
	return func() tea.Msg {
		stats, err := client.NodeStats(context.Background())
		return nodeStatsLoadedMsg{stats: stats, err: err}
	}
}

func loadNodesCmd(client *eosgrpc.Client) tea.Cmd {
	return func() tea.Msg {
		nodes, err := client.Nodes(context.Background())
		return nodesLoadedMsg{nodes: nodes, err: err}
	}
}

func loadFileSystemsCmd(client *eosgrpc.Client) tea.Cmd {
	return func() tea.Msg {
		fileSystems, err := client.FileSystems(context.Background())
		return fileSystemsLoadedMsg{fs: fileSystems, err: err}
	}
}

func loadSpacesCmd(client *eosgrpc.Client) tea.Cmd {
	return func() tea.Msg {
		spaces, err := client.Spaces(context.Background())
		return spacesLoadedMsg{spaces: spaces, err: err}
	}
}

func loadNamespaceStatsCmd(client *eosgrpc.Client) tea.Cmd {
	return func() tea.Msg {
		stats, err := client.NamespaceStats(context.Background())
		return namespaceStatsLoadedMsg{stats: stats, err: err}
	}
}

func loadDirectoryCmd(client *eosgrpc.Client, dirPath string) tea.Cmd {
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
		totalMin += allocated[i].min
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
			allocated[i].weight = 1
		}
	}

	for i := range allocated {
		if extra == 0 {
			break
		}
		share := (extra * max(allocated[i].weight, 0)) / totalWeight
		allocated[i].min += share
		extra -= share
		totalWeight -= max(allocated[i].weight, 0)
	}

	for i := range allocated {
		if extra == 0 {
			break
		}
		allocated[i].min++
		extra--
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

func entryTypeLabel(entry eosgrpc.Entry) string {
	if entry.Kind == eosgrpc.EntryKindContainer {
		return "DIR"
	}

	return "FILE"
}

func entrySize(entry eosgrpc.Entry) string {
	if entry.Kind == eosgrpc.EntryKindContainer {
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

func equivalentNodeSortValue(column int, a, b eosgrpc.NodeRecord) bool {
	switch nodeSortColumn(column) {
	case nodeSortType:
		return a.Type == b.Type
	case nodeSortHostPort:
		return a.HostPort == b.HostPort
	case nodeSortStatus:
		return a.Status == b.Status
	case nodeSortGeotag:
		return a.Geotag == b.Geotag
	case nodeSortActivated:
		return a.Activated == b.Activated
	case nodeSortFileSystems:
		return a.FileSystemCount == b.FileSystemCount
	case nodeSortHeartbeat:
		return a.HeartbeatDelta == b.HeartbeatDelta
	default:
		return a.HostPort == b.HostPort
	}
}

func equivalentFileSystemSortValue(column int, a, b eosgrpc.FileSystemRecord) bool {
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
