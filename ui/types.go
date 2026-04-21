package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	"github.com/lobis/eos-tui/eos"
)

const refreshInterval = 5 * time.Second

type viewID int

const (
	viewMGM viewID = iota
	viewQDB
	viewFST
	viewFileSystems
	viewNamespace
	viewSpaces
	viewNamespaceStats
	viewSpaceStatus // deprecated: kept for persisted-state migration only
	viewIOShaping
	viewGroups
)

const viewCount = 10

type viewTab struct {
	key   string
	label string
	view  viewID
}

var orderedViewTabs = []viewTab{
	{key: "1", label: "1 Stats", view: viewNamespaceStats},
	{key: "2", label: "2 FST", view: viewFST},
	{key: "3", label: "3 FS", view: viewFileSystems},
	{key: "4", label: "4 Namespace", view: viewNamespace},
	{key: "5", label: "5 Spaces", view: viewSpaces},
	{key: "6", label: "6 IO Traffic", view: viewIOShaping},
	{key: "7", label: "7 Groups", view: viewGroups},
	{key: "8", label: "8 MGM", view: viewMGM},
	{key: "9", label: "9 QDB", view: viewQDB},
}

func defaultActiveView() viewID {
	return viewNamespaceStats
}

func nextOrderedView(current viewID, delta int) viewID {
	for i, tab := range orderedViewTabs {
		if tab.view == current {
			next := (i + delta + len(orderedViewTabs)) % len(orderedViewTabs)
			return orderedViewTabs[next].view
		}
	}
	return defaultActiveView()
}

func viewForHotkey(key string) (viewID, bool) {
	for _, tab := range orderedViewTabs {
		if tab.key == key {
			return tab.view, true
		}
	}
	return 0, false
}

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

type namespaceAttrsLoadedMsg struct {
	path  string
	attrs []eos.NamespaceAttr
	err   error
}

type namespaceAttrSetResultMsg struct {
	path      string
	recursive bool
	err       error
}

type spaceStatusLoadedMsg struct {
	space   string
	records []eos.SpaceStatusRecord
	err     error
}

type spaceConfigResultMsg struct {
	space string
	err   error
}

type groupSetResultMsg struct {
	group  string
	status string
	err    error
	batch  bool
	count  int
	failed []string
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

type ioShapingPolicyResultMsg struct {
	id  string
	op  string
	err error
}

type eosVersionLoadedMsg struct {
	version string
}

type logLoadedMsg struct {
	filePath string
	lines    []string
	err      error
}

type commandHistoryLoadedMsg struct {
	filePath string
	lines    []string
	err      error
}

type ioShapingTickMsg struct{}
type ioShapingPolicyTickMsg struct{}
type commandLogTickMsg struct{}
type logTickMsg struct{}
type splashTickMsg struct{}

type tickMsg time.Time

type fsConfigStatusResultMsg struct {
	err error
}

type fsConfigStatusBatchResultMsg struct {
	value     string
	attempted int
	failed    []string
}

type apollonDrainResultMsg struct {
	fsID     uint64
	instance string
	output   string
	err      error
}

type errorAlert struct {
	active  bool
	fatal   bool // if true, any keypress quits the program instead of dismissing
	message string
}

type eosCheckResultMsg struct {
	err error
}

type fsConfigStatusEdit struct {
	active   bool
	fsID     uint64
	fsPath   string
	current  string
	selected int // index into configStatusOptions
	applyAll bool
	targets  []fileSystemTarget
	confirm  bool
	button   buttonID
}

type apollonDrainConfirm struct {
	active   bool
	fsID     uint64
	fsPath   string
	instance string
	command  string
	button   buttonID
}

type groupDrainConfirm struct {
	active   bool
	group    string
	current  string
	selected int
	applyAll bool
	targets  []string
	confirm  bool
	button   buttonID
}

var configStatusOptions = []string{"rw", "ro", "drain", "empty"}
var groupStatusOptions = []string{"on", "drain", "off"}

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

type fileSystemTarget struct {
	id   uint64
	path string
}

type spaceStatusEdit struct {
	active     bool
	stage      spaceStatusEditStage
	space      string
	record     eos.SpaceStatusRecord
	input      textinput.Model
	button     buttonID
	focusInput bool
}

type namespaceAttrEditStage int

const (
	attrEditStageSelect namespaceAttrEditStage = iota
	attrEditStageInput
)

type namespaceAttrEdit struct {
	active     bool
	stage      namespaceAttrEditStage
	targetPath string
	attrs      []eos.NamespaceAttr
	selected   int
	recursive  bool
	input      textinput.Model
}

type ioShapingEditStage int

const (
	ioShapingEditStageTarget ioShapingEditStage = iota
	ioShapingEditStageSelect
	ioShapingEditStageInput
	ioShapingEditStageDeleteConfirm
)

type ioShapingEditField int

const (
	ioShapingEditFieldEnabled ioShapingEditField = iota
	ioShapingEditFieldLimitRead
	ioShapingEditFieldLimitWrite
	ioShapingEditFieldReservationRead
	ioShapingEditFieldReservationWrite
	ioShapingEditFieldApply
)

type ioShapingPolicyEdit struct {
	active           bool
	stage            ioShapingEditStage
	mode             eos.IOShapingMode
	targetID         string
	createMode       bool
	hadPolicy        bool
	enabled          bool
	limitRead        string
	limitWrite       string
	reservationRead  string
	reservationWrite string
	selected         ioShapingEditField
	input            textinput.Model
	button           buttonID
	err              string
}

type filterPopup struct {
	active    bool
	view      viewID
	column    int
	input     textinput.Model
	table     table.Model
	values    []string
	navigated bool
}

type logOverlay struct {
	active     bool
	plain      bool
	tailing    bool
	wrap       bool
	host       string // specific host to read from (empty = effective target)
	filePath   string
	source     string
	rtlogQueue string
	rtlogTag   string
	rtlogSecs  int
	title      string
	allLines   []string // raw lines from tail
	filtered   []string // lines matching current filter
	filter     string   // current grep string
	filtering  bool     // filter input is active
	vp         viewport.Model
	input      textinput.Model
	err        error
	loading    bool
}

type commandPanel struct {
	active   bool
	loading  bool
	filePath string
	lines    []string
	err      error
}

type startupSplash struct {
	active bool
	frame  int
}

type logTarget struct {
	title      string
	source     string
	host       string
	filePath   string
	rtlogQueue string
	rtlogTag   string
	rtlogSecs  int
}

type filterState struct {
	column  int
	filters map[int]string
}

const namespaceFilterQueryColumn = 0

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
type spaceFilterColumn int
type spaceSortColumn int
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
	spaceFilterName spaceFilterColumn = iota
	spaceFilterType
	spaceFilterStatus
	spaceFilterGroups
	spaceFilterFiles
	spaceFilterDirs
	spaceFilterUsage
)

const spaceSortNone spaceSortColumn = -1

const (
	spaceSortName spaceSortColumn = iota
	spaceSortType
	spaceSortStatus
	spaceSortGroups
	spaceSortFiles
	spaceSortDirs
	spaceSortUsage
)

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
	app         lipgloss.Style
	header      lipgloss.Style
	tab         lipgloss.Style
	tabActive   lipgloss.Style
	panel       lipgloss.Style
	panelDim    lipgloss.Style
	selected    lipgloss.Style
	label       lipgloss.Style
	value       lipgloss.Style
	error       lipgloss.Style
	status      lipgloss.Style
	popupTitle  lipgloss.Style
	section     lipgloss.Style
	sectionRule lipgloss.Style
	splash      lipgloss.Style
	splashDim   lipgloss.Style
	splashBox   lipgloss.Style
}

type tableColumn struct {
	title  string
	min    int
	maxw   int // 0 = no max
	weight int
	right  bool
}

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
	spaceFilter          filterState
	spaceSort            sortState
	spaceStatusActive    bool

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
	nsFilter   filterState
	nsAttrs    []eos.NamespaceAttr
	nsAttrsErr error

	nsAttrsTargetPath  string
	nsAttrsLoaded      bool
	nsAttrsLoading     bool
	nsDetailContentMax int
	nsAttrEdit         namespaceAttrEdit

	spaceStatus         []eos.SpaceStatusRecord
	spaceStatusLoading  bool
	spaceStatusErr      error
	spaceStatusSelected int
	spaceStatusTarget   string

	ioShaping         []eos.IOShapingRecord
	ioShapingPolicies []eos.IOShapingPolicyRecord
	ioShapingMode     eos.IOShapingMode
	ioShapingLoading  bool
	ioShapingErr      error
	ioShapingSelected int
	ioShapingEdit     ioShapingPolicyEdit

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
	apollon     apollonDrainConfirm
	groupDrain  groupDrainConfirm
	alert       errorAlert
	log         logOverlay
	commandLog  commandPanel
	splash      startupSplash

	styles styles
}

type ioShapingMergedRow struct {
	id      string
	traffic *eos.IOShapingRecord
	policy  *eos.IOShapingPolicyRecord
}
