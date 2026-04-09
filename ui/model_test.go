package ui

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lobis/eos-tui/eos"
)

func lineCount(s string) int {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func assertCommandPanelAnchored(t *testing.T, m model, view string) {
	t.Helper()

	if got := lineCount(view); got != m.height {
		t.Fatalf("expected rendered view to fill height %d, got %d lines", m.height, got)
	}

	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	if len(lines) == 0 || !strings.Contains(lines[len(lines)-1], "L commands") {
		t.Fatalf("expected footer to remain on the last line, got:\n%s", view)
	}

	headerHeight := lipgloss.Height(m.renderHeader())
	footerHeight := lipgloss.Height(m.renderFooter())
	middleHeight := max(0, m.height-headerHeight-footerHeight)
	availableHeight := max(4, middleHeight-2)
	_, commandHeight := m.splitMainAndCommandHeights(availableHeight)
	if commandHeight == 0 {
		t.Fatalf("expected command panel to have non-zero height")
	}

	wantTitleLine := headerHeight + (middleHeight - commandHeight) + 1
	if wantTitleLine >= len(lines) {
		t.Fatalf("expected command panel title line index %d within %d lines", wantTitleLine, len(lines))
	}
	if !strings.Contains(lines[wantTitleLine], "Recent commands") {
		t.Fatalf("expected command panel title on line %d, got %q\nfull view:\n%s", wantTitleLine+1, lines[wantTitleLine], view)
	}
}

func TestNewModelRendersStartupSplashWithoutWindowSize(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)

	view := m.View()
	for _, needle := range []string{"███████", "████████", "initializing cluster view"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected initial view to contain %q, got:\n%s", needle, view)
		}
	}
}

func TestZeroWindowSizeDoesNotEraseDefaultLayout(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 0, Height: 0})
	m = updated.(model)

	if m.width == 0 || m.height == 0 {
		t.Fatalf("expected default dimensions to be preserved, got width=%d height=%d", m.width, m.height)
	}

	view := m.View()
	if !strings.Contains(view, "initializing cluster view") {
		t.Fatalf("expected rendered view to still contain startup splash, got:\n%s", view)
	}
}

func TestStartupSplashHidesAfterInitialDataArrives(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)

	m.fstStatsLoading = false
	m.fstsLoading = false
	m.fileSystemsLoading = false
	m.spacesLoading = false
	m.nsStatsLoading = false
	m.nsLoading = false
	m.groupsLoading = false
	m.spaceStatusLoading = false
	m.ioShapingLoading = false
	m.commandLog.loading = false

	updated, _ := m.Update(splashTickMsg{})
	m = updated.(model)

	if m.splash.active {
		t.Fatalf("expected startup splash to deactivate once initial loading is complete")
	}
	view := m.View()
	if strings.Contains(view, "initializing cluster view") {
		t.Fatalf("expected startup splash to disappear after loading, got:\n%s", view)
	}
}

func TestModelRendersLoadedNodeData(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 32})
	m = updated.(model)

	updated, _ = m.Update(infraLoadedMsg{
		stats: eos.NodeStats{
			State:       "OK",
			ThreadCount: 12,
			FileCount:   100,
			DirCount:    7,
			FileDescs:   42,
		},
		fsts: []eos.FstRecord{
			{
				Type:            "nodesview",
				Host:            "host",
				Port:            1095,
				Geotag:          "local",
				Status:          "online",
				Activated:       "on",
				HeartbeatDelta:  1,
				FileSystemCount: 5,
				EOSVersion:      "5.x",
				Kernel:          "linux",
			},
		},
		fs: []eos.FileSystemRecord{
			{ID: 1, Host: "host", Path: "/data/fst.1/01", Active: "online"},
		},
	})
	m = updated.(model)

	// Switch to the FST view (tab 3) to see FST nodes.
	m.activeView = viewFST

	view := m.View()
	for _, needle := range []string{"host:1095", "FST Nodes", "Selected Node", "online"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected loaded nodes view to contain %q, got:\n%s", needle, view)
		}
	}
}

func TestModelRendersFileSystemsTab(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.activeView = viewFileSystems
	m.fileSystemsLoading = false
	m.fileSystems = []eos.FileSystemRecord{
		{
			ID:            3,
			Host:          "host",
			Port:          1095,
			Path:          "/data/fst.1/03",
			SchedGroup:    "default.0",
			Active:        "online",
			Boot:          "booted",
			ConfigStatus:  "rw",
			DrainStatus:   "nodrain",
			Geotag:        "local",
			Health:        "no smartctl",
			CapacityBytes: 1000,
			UsedBytes:     500,
		},
	}

	view := m.View()
	for _, needle := range []string{"EOS Filesystems", "/data/fst.1/03", "Selected Filesystem", "no smartctl"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected filesystems view to contain %q, got:\n%s", needle, view)
		}
	}
}

func TestViewFitsWindowHeight(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 238
	m.height = 56
	m.activeView = viewFST
	m.fstsLoading = false
	m.nodeStats = eos.NodeStats{
		State:       "OK",
		ThreadCount: 489,
		FileCount:   78,
		DirCount:    19,
		FileDescs:   553,
	}
	m.fsts = []eos.FstRecord{{
		Host:            "lobisapa-dev.cern.ch",
		Port:            1095,
		Status:          "online",
		Activated:       "on",
		Geotag:          "local",
		FileSystemCount: 5,
		EOSVersion:      "5.3.27-unknown",
		Kernel:          "6.12.0-124.38.1.el10_1.x86_64",
		ThreadCount:     78,
		CapacityBytes:   1 << 40,
		UsedBytes:       1 << 30,
		FreeBytes:       1 << 39,
	}}

	view := m.View()
	if got := lineCount(view); got > m.height {
		t.Fatalf("expected rendered view to fit height %d, got %d lines", m.height, got)
	}
}

func TestFileSystemsListScrollsToSelectedEntry(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.activeView = viewFileSystems
	m.fileSystemsLoading = false
	m.width = 120
	m.height = 18
	m.fsSelected = 18

	for i := 0; i < 30; i++ {
		m.fileSystems = append(m.fileSystems, eos.FileSystemRecord{
			ID:         uint64(i),
			Host:       "lobisapa-dev.cern.ch",
			Path:       "/data/fst.1/" + padIndex(i),
			SchedGroup: "default.0",
			Active:     "online",
		})
	}

	view := m.renderFileSystemsList(m.contentWidth(), 10)
	if !strings.Contains(view, "/data/fst.1/18") {
		t.Fatalf("expected selected filesystem to be visible in scrolled list, got:\n%s", view)
	}
	if strings.Contains(view, "/data/fst.1/00") {
		t.Fatalf("expected top rows to scroll out of view, got:\n%s", view)
	}
	if !strings.Contains(view, "[") {
		t.Fatalf("expected scroll summary to be rendered, got:\n%s", view)
	}
}

func TestFileSystemsListUsesExtraWidthForPathColumn(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.activeView = viewFileSystems
	m.fileSystemsLoading = false

	m.fileSystems = []eos.FileSystemRecord{{
		ID:         1,
		Host:       "lobisapa-dev.cern.ch",
		Path:       "/data/fst.1/this-is-a-much-longer-path-than-before",
		SchedGroup: "default.123",
		Active:     "online",
	}}

	view := m.renderFileSystemsList(180, 8)
	if !strings.Contains(view, "/data/fst.1/this-is-a-mu") {
		t.Fatalf("expected longer path content to fit when width is available, got:\n%s", view)
	}
}

func TestNarrowResizeStillFitsWindowHeight(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.activeView = viewNamespace
	m.width = 80
	m.height = 20
	m.nsLoading = false
	m.directory.Path = "/"

	for i := 0; i < 40; i++ {
		m.directory.Entries = append(m.directory.Entries, eos.Entry{
			Name: "entry-" + padIndex(i) + "-with-a-long-name",
			Path: "/entry-" + padIndex(i),
			Kind: eos.EntryKindFile,
			UID:  1000,
			GID:  1000,
		})
	}
	m.nsSelected = 25

	view := m.View()
	if got := lineCount(view); got > m.height {
		t.Fatalf("expected narrow resized view to fit height %d, got %d lines\n%s", m.height, got, view)
	}
	if !strings.Contains(view, "entry-25") {
		t.Fatalf("expected selected entry to remain visible after resize, got:\n%s", view)
	}
}

func TestNamespaceViewFitsWindowHeight(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.width = 120
	m.height = 40
	m.activeView = viewNamespace
	m.nsLoaded = true
	m.nsLoading = false
	m.directory = eos.Directory{
		Path: "/test",
		Entries: []eos.Entry{
			{Name: "file1", Kind: eos.EntryKindFile},
		},
	}

	view := m.View()
	// Total height must be m.height.
	// Since splitViewHeights is currently bugged/older, it might return fewer lines.
	got := lineCount(view)
	if got != m.height {
		// This should fail before I fix it if my assumption that it's bugged is correct.
		t.Errorf("expected view to have exactly %d lines to fill screen, got %d", m.height, got)
	}
}

func TestVisibleFSTsFilterByStatus(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.fsts = []eos.FstRecord{
		{Host: "b", Port: 1095, Status: "offline", FileSystemCount: 1},
		{Host: "a", Port: 1095, Status: "online", FileSystemCount: 5},
		{Host: "c", Port: 1095, Status: "online", FileSystemCount: 3},
	}
	// fstFilterStatus corresponds to filter column 3; we set both .column and
	// .filters so visibleFSTs() applies the filter.
	m.fstFilter.column = int(fstFilterStatus)
	m.fstFilter.filters[int(fstFilterStatus)] = "online"

	fsts := m.visibleFSTs()
	if len(fsts) != 2 {
		t.Fatalf("expected 2 filtered fsts, got %d", len(fsts))
	}
	if fsts[0].Host != "a" || fsts[1].Host != "c" {
		t.Fatalf("unexpected filtered order: %#v", fsts)
	}
}

func TestVisibleFSTsSortByFileSystemsDesc(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.fsts = []eos.FstRecord{
		{Host: "b", Port: 1095, Status: "online", FileSystemCount: 1},
		{Host: "a", Port: 1095, Status: "online", FileSystemCount: 5},
		{Host: "c", Port: 1095, Status: "online", FileSystemCount: 3},
	}
	m.fstSort.column = int(fstSortFileSystems)
	m.fstSort.desc = true

	fsts := m.visibleFSTs()
	if got := []string{fsts[0].Host, fsts[1].Host, fsts[2].Host}; strings.Join(got, ",") != "a,c,b" {
		t.Fatalf("unexpected sort order: %v", got)
	}
}

func TestVisibleFileSystemsSortByUsedDesc(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.fileSystems = []eos.FileSystemRecord{
		{ID: 1, Host: "a", Path: "/a", CapacityBytes: 100, UsedBytes: 10},
		{ID: 2, Host: "b", Path: "/b", CapacityBytes: 100, UsedBytes: 90},
		{ID: 3, Host: "c", Path: "/c", CapacityBytes: 100, UsedBytes: 50},
	}
	m.fsSort.column = int(fsSortUsed)
	m.fsSort.desc = true

	fileSystems := m.visibleFileSystems()
	if got := []uint64{fileSystems[0].ID, fileSystems[1].ID, fileSystems[2].ID}; got[0] != 2 || got[1] != 3 || got[2] != 1 {
		t.Fatalf("unexpected filesystem sort order: %v", got)
	}
}

func TestDescendingNodeSortDoesNotTreatEqualRowsAsLess(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.fstSort = sortState{column: int(fstSortNoFS), desc: true}

	node := eos.FstRecord{Host: "fst01", Port: 1095, FileSystemCount: 5}
	if m.lessNode(node, node) {
		t.Fatalf("expected identical nodes to compare equal in descending sort")
	}

	m.fsts = []eos.FstRecord{
		{Host: "fst-b", Port: 1095, FileSystemCount: 5},
		{Host: "fst-a", Port: 1095, FileSystemCount: 5},
	}
	fsts := m.visibleFSTs()
	if got := []string{fsts[0].Host, fsts[1].Host}; strings.Join(got, ",") != "fst-a,fst-b" {
		t.Fatalf("expected host tie-breaker to stay deterministic, got %v", got)
	}
}

func TestDescendingFileSystemSortDoesNotTreatEqualRowsAsLess(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.fsSort = sortState{column: int(fsSortUsed), desc: true}

	fs := eos.FileSystemRecord{ID: 1, Host: "fst01", Path: "/data/1", CapacityBytes: 100, UsedBytes: 50}
	if m.lessFileSystem(fs, fs) {
		t.Fatalf("expected identical filesystems to compare equal in descending sort")
	}

	m.fileSystems = []eos.FileSystemRecord{
		{ID: 2, Host: "fst-b", Path: "/data/2", CapacityBytes: 100, UsedBytes: 50},
		{ID: 1, Host: "fst-a", Path: "/data/1", CapacityBytes: 100, UsedBytes: 50},
	}
	fileSystems := m.visibleFileSystems()
	if got := []uint64{fileSystems[0].ID, fileSystems[1].ID}; got[0] != 1 || got[1] != 2 {
		t.Fatalf("expected ID tie-breaker to stay deterministic, got %v", got)
	}
}

func TestDescendingGroupSortDoesNotTreatEqualRowsAsLess(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.groupSort = sortState{column: int(groupSortNoFS), desc: true}

	group := eos.GroupRecord{Name: "default.0", Status: "online", NoFS: 3}
	if m.lessGroup(group, group) {
		t.Fatalf("expected identical groups to compare equal in descending sort")
	}

	m.groups = []eos.GroupRecord{
		{Name: "default.1", Status: "online", NoFS: 3},
		{Name: "default.0", Status: "online", NoFS: 3},
	}
	groups := m.visibleGroups()
	if got := []string{groups[0].Name, groups[1].Name}; strings.Join(got, ",") != "default.0,default.1" {
		t.Fatalf("expected name tie-breaker to stay deterministic, got %v", got)
	}
}

func TestFilterPopupAppliesToFSTs(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.activeView = viewFST
	m.fsts = []eos.FstRecord{
		{Host: "alpha", Port: 1095, Status: "online", FileSystemCount: 2},
		{Host: "beta", Port: 1095, Status: "offline", FileSystemCount: 1},
	}
	m.fstColumnSelected = int(fstFilterHost)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(model)
	if !m.popup.active {
		t.Fatalf("expected filter popup to open")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)

	fsts := m.visibleFSTs()
	if len(fsts) != 1 || fsts[0].Host != "beta" {
		t.Fatalf("expected filter input to keep beta only, got %#v", fsts)
	}
}

func TestFSTsRenderBeforeStatsArrive(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 24
	m.activeView = viewFST

	updated, _ := m.Update(fstsLoadedMsg{
		fsts: []eos.FstRecord{
			{Host: "fast-node", Port: 1095, Status: "online", Activated: "on", Geotag: "local", FileSystemCount: 5},
		},
	})
	m = updated.(model)

	view := m.View()
	if !strings.Contains(view, "fast-node:1095") {
		t.Fatalf("expected node table to render before stats load, got:\n%s", view)
	}
	if strings.Contains(view, "Loading cluster summary...") {
		t.Fatalf("expected cluster summary loading to stay out of the FST view, got:\n%s", view)
	}
}

func TestNewModelDoesNotEagerLoadNamespace(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)

	if m.nsLoading {
		t.Fatalf("expected namespace loading to be lazy at startup")
	}
	if m.nsLoaded {
		t.Fatalf("expected namespace to start unloaded")
	}
}

func TestSwitchingToNamespaceStartsLazyLoad(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)

	// Key '4' switches to the Namespace view in the current layout.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	m = updated.(model)

	if !m.nsLoading {
		t.Fatalf("expected namespace load to start when switching to namespace view")
	}
	if cmd == nil {
		t.Fatalf("expected namespace switch to return a load command")
	}
}

func TestFSTSortCyclesOnSelectedColumn(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.activeView = viewFST
	m.fstColumnSelected = int(fstFilterNoFS)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	m = updated.(model)
	if m.fstSort.column != int(fstSortNoFS) || m.fstSort.desc {
		t.Fatalf("expected first sort press to set ascending nofs sort, got %+v", m.fstSort)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	m = updated.(model)
	if m.fstSort.column != int(fstSortNoFS) || !m.fstSort.desc {
		t.Fatalf("expected second sort press to set descending nofs sort, got %+v", m.fstSort)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	m = updated.(model)
	if m.fstSort.column != int(fstSortNone) {
		t.Fatalf("expected third sort press to clear sort, got %+v", m.fstSort)
	}
}

func TestFSTEnumFilterCyclesOnSelectedColumn(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.activeView = viewFST
	m.fsts = []eos.FstRecord{
		{Host: "a", Port: 1095, Status: "online", FileSystemCount: 1},
		{Host: "b", Port: 1095, Status: "offline", FileSystemCount: 1},
	}
	m.fstColumnSelected = int(fstFilterStatus)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(model)
	if !m.popup.active {
		t.Fatalf("expected popup to open for enum filter")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o', 'f', 'f'}})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.fstFilter.column != int(fstFilterStatus) || m.fstFilter.filters[int(fstFilterStatus)] != "offline" {
		t.Fatalf("expected enum popup selection to apply offline filter, got %+v", m.fstFilter)
	}
}

func TestHeaderShowsSelectedAndSortedColumn(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.fstsLoading = false
	// Display column 0 = "host".
	m.fstColumnSelected = 0
	m.fstSort = sortState{column: 0} // column-0 indicator appears on "host"
	m.fsts = []eos.FstRecord{{Host: "a", Port: 1095, FileSystemCount: 1}}

	view := m.renderNodesList(m.contentWidth(), 10)
	if !strings.Contains(view, "[host") || !strings.Contains(view, "host↑") {
		t.Fatalf("expected header to show selected sorted column, got:\n%s", view)
	}
}

func TestFilterPopupCanBeCancelled(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.activeView = viewFST
	m.fsts = []eos.FstRecord{{Host: "alpha", Port: 1095, FileSystemCount: 1}}
	m.fstColumnSelected = int(fstFilterHost)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(model)
	if !m.popup.active {
		t.Fatalf("expected popup to open")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(model)
	if m.popup.active {
		t.Fatalf("expected popup to close on escape")
	}
	if m.fstFilter.filters[int(fstFilterHost)] != "" {
		t.Fatalf("expected filter to remain unchanged after cancel, got %+v", m.fstFilter)
	}
}

// TestColumnHeadersUseConsistentStyle verifies that all column header rows in
// all views use m.styles.label (muted blue), not m.styles.header (bold green).
// The header style is reserved for the application title bar only.
//
// This test guards against the regression where new render functions accidentally
// call m.styles.header.Render(formatTableRow(...)) instead of going through
// renderSimpleHeaderRow or renderSelectableHeaderRow.
func TestColumnHeadersUseConsistentStyle(t *testing.T) {
	m := NewModel(nil, "test", "/").(model)
	m.width = 180
	m.height = 40

	// Populate enough data so every view renders at least one header row.
	m.fsts = []eos.FstRecord{{Host: "fst", Port: 1095, Status: "online", Activated: "on", FileSystemCount: 1}}
	m.fstsLoading = false
	m.mgms = []eos.MgmRecord{{Host: "mgm", Port: 1094, QDBHost: "mgm", QDBPort: 7777, Role: "leader", Status: "online", EOSVersion: "5.x"}}
	m.mgmsLoading = false
	m.eosVersion = "5.x"
	m.fileSystems = []eos.FileSystemRecord{{ID: 1, Host: "h", Path: "/p", Active: "online"}}
	m.fileSystemsLoading = false
	m.spaces = []eos.SpaceRecord{{Name: "default", Type: "groupbalancer"}}
	m.spacesLoading = false
	m.spaceStatus = []eos.SpaceStatusRecord{{Key: "k", Value: "v"}}
	m.spaceStatusLoading = false
	m.ioShaping = []eos.IOShapingRecord{{ID: "app1", Type: "app"}}
	m.ioShapingLoading = false
	m.directory.Entries = []eos.Entry{{Name: "f", Path: "/f", Kind: eos.EntryKindFile}}
	m.nsLoaded = true
	m.groups = []eos.GroupRecord{{Name: "default.0", Status: "online", NoFS: 3}}
	m.groupsLoading = false

	// headerStyle is the color used by m.styles.header (bold green, color 86).
	// We detect it by rendering a known string with that style.
	headerStyleMarker := m.styles.header.Render("X")

	// labelStyleMarker is what we expect column headers to look like.
	labelStyleMarker := m.styles.label.Render("X")

	// They must be visually distinct for this test to be meaningful.
	if headerStyleMarker == labelStyleMarker {
		t.Skip("header and label styles produce identical output in this terminal; skipping")
	}

	viewsToCheck := []struct {
		name string
		view viewID
	}{
		{"MGM", viewMGM},
		{"QDB", viewQDB},
		{"FST", viewFST},
		{"FS", viewFileSystems},
		{"Spaces", viewSpaces},
		{"SpaceStatus", viewSpaceStatus},
		{"IOTraffic", viewIOShaping},
		{"Groups", viewGroups},
	}

	for _, tc := range viewsToCheck {
		m.activeView = tc.view
		rendered := m.renderBody(30)

		// The app-title bold-green style must NOT appear inside a body view.
		if strings.Contains(rendered, headerStyleMarker) {
			t.Errorf("view %s: column headers use m.styles.header (app-title style); use renderSimpleHeaderRow or renderSelectableHeaderRow instead", tc.name)
		}
		// The label style MUST appear (at least the column header row).
		if !strings.Contains(rendered, labelStyleMarker) {
			t.Errorf("view %s: expected column headers styled with m.styles.label, but label style not found", tc.name)
		}
	}
}

func TestSwitchingToSpacesTriggersLoad(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	// Spaces start with loading=true from init but no data yet.
	// Simulate: loading finished with no data (first tick cleared it).
	m.spacesLoading = false
	m.spaces = nil
	m.spacesErr = nil

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	m = updated.(model)

	if !m.spacesLoading {
		t.Fatalf("expected spacesLoading=true after switching to spaces view")
	}
	if cmd == nil {
		t.Fatalf("expected a load command to be returned when switching to spaces view")
	}
}

func TestSwitchingToNsStatsTriggersLoad(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.nsStatsLoading = false
	m.namespaceStats = eos.NamespaceStats{}
	m.nsStatsErr = nil
	m.fstStatsLoading = false
	m.nodeStats = eos.NodeStats{}
	m.nodeStatsErr = nil

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = updated.(model)

	if !m.nsStatsLoading {
		t.Fatalf("expected nsStatsLoading=true after switching to namespace stats view")
	}
	if !m.fstStatsLoading {
		t.Fatalf("expected fstStatsLoading=true after switching to namespace stats view")
	}
	if cmd == nil {
		t.Fatalf("expected a load command to be returned when switching to namespace stats view")
	}
}

func TestSpacesLoadedMsgUpdatesModel(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.spacesLoading = true

	updated, _ := m.Update(spacesLoadedMsg{
		spaces: []eos.SpaceRecord{
			{Name: "default", Type: "groupbalancer", CapacityBytes: 1 << 40},
			{Name: "ecbench", Type: "groupbalancer", CapacityBytes: 2 << 40},
		},
	})
	m = updated.(model)

	if m.spacesLoading {
		t.Fatal("expected spacesLoading=false after spacesLoadedMsg")
	}
	if len(m.spaces) != 2 {
		t.Fatalf("expected 2 spaces, got %d", len(m.spaces))
	}
	if m.spaces[0].Name != "default" {
		t.Errorf("expected first space to be 'default', got %q", m.spaces[0].Name)
	}
}

func TestNamespaceStatsLoadedMsgUpdatesModel(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.nsStatsLoading = true

	updated, _ := m.Update(namespaceStatsLoadedMsg{
		stats: eos.NamespaceStats{
			TotalFiles:       78,
			TotalDirectories: 19,
			CurrentFID:       7661,
		},
	})
	m = updated.(model)

	if m.nsStatsLoading {
		t.Fatal("expected nsStatsLoading=false after namespaceStatsLoadedMsg")
	}
	if m.namespaceStats.TotalFiles != 78 {
		t.Errorf("expected TotalFiles=78, got %d", m.namespaceStats.TotalFiles)
	}
	if m.namespaceStats.TotalDirectories != 19 {
		t.Errorf("expected TotalDirectories=19, got %d", m.namespaceStats.TotalDirectories)
	}
}

func TestSpacesViewRendersWithData(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 30
	m.activeView = viewSpaces
	m.spacesLoading = false
	m.spaces = []eos.SpaceRecord{
		{Name: "default", Type: "groupbalancer", CapacityBytes: 1 << 40, UsedBytes: 1 << 38},
		{Name: "ecbench", Type: "groupbalancer", CapacityBytes: 2 << 40, UsedBytes: 1 << 39},
	}

	view := m.View()
	for _, needle := range []string{"EOS Spaces", "default", "ecbench", "groupbalancer"} {
		if !strings.Contains(view, needle) {
			t.Errorf("expected spaces view to contain %q, got:\n%s", needle, view)
		}
	}
}

func TestNamespaceStatsViewRendersWithData(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 30
	m.activeView = viewNamespaceStats
	m.nsStatsLoading = false
	m.fstStatsLoading = false
	m.nodeStats = eos.NodeStats{
		State:       "OK",
		ThreadCount: 489,
		FileCount:   78,
		DirCount:    19,
		FileDescs:   553,
	}
	m.namespaceStats = eos.NamespaceStats{
		TotalFiles:       78,
		TotalDirectories: 19,
		CurrentFID:       7661,
		CurrentCID:       1234,
		MasterHost:       "mgm01:1094",
	}

	view := m.View()
	for _, needle := range []string{"General Statistics", "Cluster Summary", "Namespace Statistics", "Master", "489", "78", "19"} {
		if !strings.Contains(view, needle) {
			t.Errorf("expected namespace stats view to contain %q, got:\n%s", needle, view)
		}
	}
}

func TestNamespaceStatsViewCanRenderNamespaceStatsBeforeClusterSummaryArrives(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 30
	m.activeView = viewNamespaceStats
	m.nsStatsLoading = false
	m.fstStatsLoading = true
	m.namespaceStats = eos.NamespaceStats{
		MasterHost:       "mgm01:1094",
		TotalFiles:       78,
		TotalDirectories: 19,
	}
	m.splash.active = false

	view := m.View()
	if !strings.Contains(view, "Loading cluster summary...") {
		t.Fatalf("expected general stats view to keep a cluster-summary loading section, got:\n%s", view)
	}
	for _, needle := range []string{"Namespace Statistics", "mgm01:1094", "78", "19"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected general stats view to still show namespace data %q, got:\n%s", needle, view)
		}
	}
}

func TestSpacesViewShowsLoadingState(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 30
	m.activeView = viewSpaces
	m.spacesLoading = true
	m.splash.active = false

	view := m.View()
	if !strings.Contains(view, "Loading") {
		t.Errorf("expected spaces view to show loading state, got:\n%s", view)
	}
}

func TestNamespaceStatsViewShowsLoadingState(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 30
	m.activeView = viewNamespaceStats
	m.nsStatsLoading = true
	m.splash.active = false

	view := m.View()
	if !strings.Contains(view, "Loading") {
		t.Errorf("expected namespace stats view to show loading state, got:\n%s", view)
	}
}

func TestEOSVersionLoadedMsgUpdatesModel(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)

	updated, _ := m.Update(eosVersionLoadedMsg{version: "5.3.27"})
	m = updated.(model)

	if m.eosVersion != "5.3.27" {
		t.Errorf("expected eosVersion=5.3.27, got %q", m.eosVersion)
	}
}

func TestEOSVersionLoadedMsgIgnoresEmpty(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.eosVersion = "5.3.27"

	updated, _ := m.Update(eosVersionLoadedMsg{version: ""})
	m = updated.(model)

	// Empty version must not overwrite existing value.
	if m.eosVersion != "5.3.27" {
		t.Errorf("expected existing eosVersion to be preserved, got %q", m.eosVersion)
	}
}

func TestSpacesAndNsStatsAreInInitialLoadBatch(t *testing.T) {
	// Spaces and NS Stats should start as loading=true because loadInfraCmd
	// fetches them at startup. This test verifies the initial model state
	// reflects that intent (they should not start as loaded/false).
	m := NewModel(nil, "local eos cli", "/").(model)

	if !m.spacesLoading {
		t.Error("expected spacesLoading=true at startup (infra batch fetches spaces)")
	}
	if !m.nsStatsLoading {
		t.Error("expected nsStatsLoading=true at startup (infra batch fetches ns stats)")
	}
}

// ---- selectedHostForView tests ---------------------------------------------

func TestSelectedHostForViewFST(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.activeView = viewFST
	m.fsts = []eos.FstRecord{
		{Host: "fst01.cern.ch", Port: 1095, Status: "online", Type: "fst"},
		{Host: "fst02.cern.ch", Port: 1095, Status: "online", Type: "fst"},
	}
	m.fstSelected = 1

	got := m.selectedHostForView()
	if got != "fst02.cern.ch" {
		t.Errorf("expected fst02.cern.ch, got %q", got)
	}
}

func TestSelectedHostForViewMGM(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.activeView = viewMGM
	m.mgms = []eos.MgmRecord{
		{Host: "mgm01.cern.ch", Port: 1094, Role: "leader"},
		{Host: "mgm02.cern.ch", Port: 1094, Role: "follower"},
	}
	m.mgmSelected = 0

	got := m.selectedHostForView()
	if got != "mgm01.cern.ch" {
		t.Errorf("expected mgm01.cern.ch, got %q", got)
	}
}

func TestSelectedHostForViewQDB(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.activeView = viewQDB
	m.mgms = []eos.MgmRecord{
		{QDBHost: "qdb01.cern.ch", QDBPort: 7777, Role: "leader"},
		{QDBHost: "qdb02.cern.ch", QDBPort: 7777, Role: "follower"},
	}
	m.qdbSelected = 1

	got := m.selectedHostForView()
	if got != "qdb02.cern.ch" {
		t.Errorf("expected qdb02.cern.ch, got %q", got)
	}
}

func TestSelectedHostForViewFileSystems(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.activeView = viewFileSystems
	m.fileSystems = []eos.FileSystemRecord{
		{ID: 1, Host: "fst01.cern.ch", Path: "/data/01"},
		{ID: 2, Host: "fst02.cern.ch", Path: "/data/02"},
	}
	m.fsSelected = 0

	got := m.selectedHostForView()
	if got != "fst01.cern.ch" {
		t.Errorf("expected fst01.cern.ch, got %q", got)
	}
}

func TestSelectedHostForViewNoSelection(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.activeView = viewSpaces // no host concept

	got := m.selectedHostForView()
	if got != "" {
		t.Errorf("expected empty string for spaces view, got %q", got)
	}
}

func TestSelectedHostForViewEmptyList(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.activeView = viewFST
	m.fsts = nil
	m.fstSelected = 0

	got := m.selectedHostForView()
	if got != "" {
		t.Errorf("expected empty for empty FST list, got %q", got)
	}
}

// ---- MGM/QDB navigation tests ----------------------------------------------

func TestMGMNavigationUpDown(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.activeView = viewMGM
	m.mgms = []eos.MgmRecord{
		{Host: "mgm01", Port: 1094, Role: "leader"},
		{Host: "mgm02", Port: 1094, Role: "follower"},
		{Host: "mgm03", Port: 1094, Role: "follower"},
	}
	m.mgmSelected = 0

	// Navigate down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(model)
	if m.mgmSelected != 1 {
		t.Fatalf("expected mgmSelected=1 after j, got %d", m.mgmSelected)
	}

	// Navigate down again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(model)
	if m.mgmSelected != 2 {
		t.Fatalf("expected mgmSelected=2 after j, got %d", m.mgmSelected)
	}

	// Navigate down at end — should clamp
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(model)
	if m.mgmSelected != 2 {
		t.Fatalf("expected mgmSelected=2 (clamped), got %d", m.mgmSelected)
	}

	// Navigate up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(model)
	if m.mgmSelected != 1 {
		t.Fatalf("expected mgmSelected=1 after k, got %d", m.mgmSelected)
	}
}

func TestMGMNavigationGG(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.activeView = viewMGM
	m.mgms = []eos.MgmRecord{
		{Host: "mgm01", Port: 1094},
		{Host: "mgm02", Port: 1094},
		{Host: "mgm03", Port: 1094},
	}
	m.mgmSelected = 1

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(model)
	if m.mgmSelected != 2 {
		t.Fatalf("expected G to go to last, got %d", m.mgmSelected)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(model)
	if m.mgmSelected != 0 {
		t.Fatalf("expected g to go to first, got %d", m.mgmSelected)
	}
}

func TestQDBNavigationUpDown(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.activeView = viewQDB
	m.mgms = []eos.MgmRecord{
		{QDBHost: "qdb01", QDBPort: 7777, Role: "leader"},
		{QDBHost: "qdb02", QDBPort: 7777, Role: "follower"},
	}
	m.qdbSelected = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)
	if m.qdbSelected != 1 {
		t.Fatalf("expected qdbSelected=1 after down, got %d", m.qdbSelected)
	}

	// Should not go beyond list end
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)
	if m.qdbSelected != 1 {
		t.Fatalf("expected qdbSelected clamped at 1, got %d", m.qdbSelected)
	}
}

func TestMGMViewShowsSelectedRow(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.width = 120
	m.height = 30
	m.activeView = viewMGM
	m.mgmsLoading = false
	m.mgms = []eos.MgmRecord{
		{Host: "mgm01.cern.ch", Port: 1094, Role: "leader", Status: "online"},
		{Host: "mgm02.cern.ch", Port: 1094, Role: "follower", Status: "online"},
	}
	m.mgmSelected = 0

	view := m.View()
	if !strings.Contains(view, "mgm01.cern.ch") {
		t.Errorf("expected mgm01 in view, got:\n%s", view)
	}
	if !strings.Contains(view, "mgm02.cern.ch") {
		t.Errorf("expected mgm02 in view, got:\n%s", view)
	}
}

func TestQDBViewShowsSelectedRow(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.width = 120
	m.height = 30
	m.activeView = viewQDB
	m.mgmsLoading = false
	m.mgms = []eos.MgmRecord{
		{QDBHost: "qdb01.cern.ch", QDBPort: 7777, Role: "leader", Status: "online", EOSVersion: "5.3.29"},
		{QDBHost: "qdb02.cern.ch", QDBPort: 7777, Role: "follower", Status: "online", EOSVersion: "5.3.29"},
	}
	m.qdbSelected = 1

	view := m.View()
	for _, needle := range []string{"qdb01.cern.ch", "qdb02.cern.ch", "leader", "follower"} {
		if !strings.Contains(view, needle) {
			t.Errorf("expected %q in QDB view, got:\n%s", needle, view)
		}
	}
}

func TestMGMSelectedHostChangesWithNavigation(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.activeView = viewMGM
	m.mgms = []eos.MgmRecord{
		{Host: "mgm01.cern.ch", Port: 1094},
		{Host: "mgm02.cern.ch", Port: 1094},
	}
	m.mgmSelected = 0

	if got := m.selectedHostForView(); got != "mgm01.cern.ch" {
		t.Fatalf("expected mgm01.cern.ch initially, got %q", got)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)

	if got := m.selectedHostForView(); got != "mgm02.cern.ch" {
		t.Fatalf("expected mgm02.cern.ch after navigating down, got %q", got)
	}
}

func padIndex(i int) string {
	if i < 10 {
		return "0" + strconv.Itoa(i)
	}
	return strconv.Itoa(i)
}

// TestFSTFilterColumnAlignment verifies that every navigable column index
// (0 .. nodeColumnCount-1) maps to the correct field in fstFilterValueForColumn.
// This is a regression test for the off-by-one bug where fstFilterType=0 sat
// before fstFilterHostPort in the iota, causing visual column [hostport] (i=0)
// to actually filter by the Type field.
func TestFSTFilterColumnAlignment(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)

	node := eos.FstRecord{
		Host:            "fst01.cern.ch",
		Port:            1095,
		Geotag:          "cern::prod",
		Status:          "online",
		Activated:       "on",
		HeartbeatDelta:  3,
		FileSystemCount: 4,
		EOSVersion:      "5.2.7",
		Type:            "fst",
	}

	cases := []struct {
		column  fstFilterColumn
		label   string
		wantVal string
	}{
		{fstFilterHost, "host", node.Host},
		{fstFilterPort, "port", "1095"},
		{fstFilterGeotag, "geotag", node.Geotag},
		{fstFilterStatus, "status", node.Status},
		{fstFilterActivated, "activated", node.Activated},
		{fstFilterHeartbeatDelta, "heartbeatdelta", "3"},
		{fstFilterNoFS, "nofs", "4"},
		{fstFilterEOSVersion, "eos version", node.EOSVersion},
	}

	for _, tc := range cases {
		col := int(tc.column)

		// Each navigable column index must be within nodeColumnCount.
		if col >= nodeColumnCount() {
			t.Errorf("column %s (index %d) is >= nodeColumnCount (%d)", tc.label, col, nodeColumnCount())
		}

		// fstFilterValueForColumn must return the correct field.
		got := m.fstFilterValueForColumn(node, col)
		if got != tc.wantVal {
			t.Errorf("column %s (index %d): fstFilterValueForColumn = %q, want %q", tc.label, col, got, tc.wantVal)
		}

		// fstFilterColumnLabel must return the right label for the column.
		m.fstFilter.column = col
		gotLabel := m.fstFilterColumnLabel()
		if gotLabel != tc.label {
			t.Errorf("column index %d: fstFilterColumnLabel = %q, want %q", col, gotLabel, tc.label)
		}
	}

	// fstFilterType must NOT be navigable (its index must be >= nodeColumnCount).
	if int(fstFilterType) < nodeColumnCount() {
		t.Errorf("fstFilterType index %d is navigable (< nodeColumnCount %d); it should be hidden", fstFilterType, nodeColumnCount())
	}
}

// TestFSTFilterAppliesCorrectField sets a filter on the hostport column and
// verifies that only the hostport field is used for matching (not the type field).
func TestFSTFilterAppliesCorrectField(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)

	m.fsts = []eos.FstRecord{
		{Host: "alpha.cern.ch", Port: 1095, Type: "fst", FileSystemCount: 1},
		{Host: "beta.cern.ch", Port: 1095, Type: "fst", FileSystemCount: 1},
		{Host: "gamma.cern.ch", Port: 1095, Type: "fst", FileSystemCount: 1},
	}

	// Filter host column for "alpha".
	m.fstFilter.filters = map[int]string{int(fstFilterHost): "alpha"}

	visible := m.visibleFSTs()
	if len(visible) != 1 {
		t.Fatalf("expected 1 visible FST after hostport filter, got %d", len(visible))
	}
	if visible[0].Host != "alpha.cern.ch" {
		t.Errorf("expected alpha.cern.ch, got %q", visible[0].Host)
	}
}

// TestIOShapingMergedRowsIncludesPolicyOnly verifies that ioShapingMergedRows
// returns rows for policy-only entries (no current traffic) as well as
// traffic-only and combined entries — and that the result is sorted by id.
func TestIOShapingMergedRowsIncludesPolicyOnly(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.ioShapingMode = eos.IOShapingApps

	m.ioShaping = []eos.IOShapingRecord{
		{ID: "app-traffic", Type: "app", ReadBps: 1e6},
		{ID: "app-both", Type: "app", WriteBps: 2e6},
	}
	m.ioShapingPolicies = []eos.IOShapingPolicyRecord{
		{ID: "app-both", Type: "app", Enabled: true, LimitWriteBytesPerSec: 5e9},
		{ID: "app-policy-only", Type: "app", Enabled: false, LimitWriteBytesPerSec: 1e9},
	}

	rows := m.ioShapingMergedRows()

	if len(rows) != 3 {
		t.Fatalf("expected 3 merged rows (traffic-only, both, policy-only), got %d", len(rows))
	}

	// Rows must be sorted alphabetically by id.
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.id
	}
	for i := 1; i < len(ids); i++ {
		if ids[i] < ids[i-1] {
			t.Errorf("rows not sorted: %v", ids)
			break
		}
	}

	byID := make(map[string]ioShapingMergedRow, len(rows))
	for _, r := range rows {
		byID[r.id] = r
	}

	// app-traffic: traffic present, no policy.
	if r := byID["app-traffic"]; r.traffic == nil || r.policy != nil {
		t.Errorf("app-traffic: expected traffic=set policy=nil, got traffic=%v policy=%v", r.traffic, r.policy)
	}
	// app-both: both traffic and policy present.
	if r := byID["app-both"]; r.traffic == nil || r.policy == nil {
		t.Errorf("app-both: expected traffic=set policy=set, got traffic=%v policy=%v", r.traffic, r.policy)
	}
	// app-policy-only: policy present, no traffic.
	if r := byID["app-policy-only"]; r.traffic != nil || r.policy == nil {
		t.Errorf("app-policy-only: expected traffic=nil policy=set, got traffic=%v policy=%v", r.traffic, r.policy)
	}
}

// TestIOShapingNavigationIncludesPolicyOnlyRows verifies that the navigation
// bound (n) used in updateIOShapingKeys accounts for policy-only rows so the
// user can navigate to them with j/down.
func TestIOShapingNavigationIncludesPolicyOnlyRows(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.activeView = viewIOShaping
	m.ioShapingMode = eos.IOShapingApps

	// One traffic record, one policy-only record → merged count = 2.
	m.ioShaping = []eos.IOShapingRecord{
		{ID: "app-a", Type: "app"},
	}
	m.ioShapingPolicies = []eos.IOShapingPolicyRecord{
		{ID: "app-b", Type: "app", Enabled: true},
	}

	if got := len(m.ioShapingMergedRows()); got != 2 {
		t.Fatalf("expected 2 merged rows, got %d", got)
	}

	// Simulate pressing "down" from row 0 — should reach row 1 (policy-only).
	m.ioShapingSelected = 0
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	updated, _ := m.Update(msg)
	m2 := updated.(model)

	if m2.ioShapingSelected != 1 {
		t.Errorf("after pressing j, expected ioShapingSelected=1, got %d", m2.ioShapingSelected)
	}
}

// ---- Groups view tests -----------------------------------------------------

func TestGroupsViewRendersWithData(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 30
	m.activeView = viewGroups
	m.groupsLoading = false
	m.groups = []eos.GroupRecord{
		{Name: "default.0", Status: "online", NoFS: 3, CapacityBytes: 1 << 40, UsedBytes: 1 << 38},
		{Name: "default.1", Status: "offline", NoFS: 2, CapacityBytes: 2 << 40, UsedBytes: 1 << 39},
	}

	view := m.View()
	for _, needle := range []string{"EOS Groups", "default.0", "default.1", "online", "offline", "8 Groups", "Selected Group", "Free"} {
		if !strings.Contains(view, needle) {
			t.Errorf("expected groups view to contain %q, got:\n%s", needle, view)
		}
	}
}

func TestGroupsViewFitsWindowHeight(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 30
	m.activeView = viewGroups
	m.groupsLoading = false
	m.groups = make([]eos.GroupRecord, 50)
	for i := range m.groups {
		m.groups[i] = eos.GroupRecord{
			Name:   "default." + strconv.Itoa(i),
			Status: "online",
			NoFS:   3,
		}
	}

	view := m.View()
	if got := lineCount(view); got > m.height {
		t.Fatalf("expected groups view to fit height %d, got %d lines", m.height, got)
	}
}

func TestGroupsViewFillsWindowHeightExactly(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 30
	m.activeView = viewGroups
	m.groupsLoading = false
	m.groups = []eos.GroupRecord{
		{Name: "default.0", Status: "online", NoFS: 3},
	}

	view := m.View()
	if got := lineCount(view); got != m.height {
		t.Fatalf("expected groups view to fill height %d, got %d lines", m.height, got)
	}
}

func TestGroupsViewShowsLoadingState(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 30
	m.activeView = viewGroups
	m.groupsLoading = true
	m.splash.active = false

	view := m.View()
	if !strings.Contains(view, "Loading") {
		t.Errorf("expected groups view to show loading state, got:\n%s", view)
	}
}

func TestGroupsLoadedMsgUpdatesModel(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.groupsLoading = true

	updated, _ := m.Update(groupsLoadedMsg{
		groups: []eos.GroupRecord{
			{Name: "default.0", Status: "online", NoFS: 3},
			{Name: "default.1", Status: "online", NoFS: 2},
		},
	})
	m = updated.(model)

	if m.groupsLoading {
		t.Fatal("expected groupsLoading=false after groupsLoadedMsg")
	}
	if len(m.groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(m.groups))
	}
}

func TestSwitchingToGroupsTriggersLoad(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.groupsLoading = false
	m.groups = nil
	m.groupsErr = nil

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'8'}})
	m = updated.(model)

	if m.activeView != viewGroups {
		t.Fatalf("expected activeView=viewGroups after pressing 8, got %d", m.activeView)
	}
	if !m.groupsLoading {
		t.Fatalf("expected groupsLoading=true after switching to groups view")
	}
	if cmd == nil {
		t.Fatalf("expected a load command to be returned when switching to groups view")
	}
}

func TestHotkeysFollowNewViewOrdering(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)

	cases := []struct {
		key  rune
		want viewID
	}{
		{'1', viewNamespaceStats},
		{'2', viewFST},
		{'3', viewFileSystems},
		{'4', viewNamespace},
		{'5', viewSpaces},
		{'6', viewSpaceStatus},
		{'7', viewIOShaping},
		{'8', viewGroups},
		{'9', viewMGM},
		{'0', viewQDB},
	}

	for _, tc := range cases {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tc.key}})
		m = updated.(model)
		if m.activeView != tc.want {
			t.Fatalf("expected key %q to switch to view %d, got %d", string(tc.key), tc.want, m.activeView)
		}
	}
}

func TestTabCyclesThroughNewViewOrdering(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.activeView = viewNamespaceStats

	expected := []viewID{
		viewFST,
		viewFileSystems,
		viewNamespace,
		viewSpaces,
		viewSpaceStatus,
		viewIOShaping,
		viewGroups,
		viewMGM,
		viewQDB,
		viewNamespaceStats,
	}

	for _, want := range expected {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = updated.(model)
		if m.activeView != want {
			t.Fatalf("expected tab cycle to reach view %d, got %d", want, m.activeView)
		}
	}
}

func TestNamespaceDetailsRenderAttributes(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.width = 120
	m.height = 28
	m.activeView = viewNamespace
	m.nsLoaded = true
	m.nsLoading = false
	m.directory = eos.Directory{
		Path: "/eos/dev",
		Self: eos.Entry{Name: "dev", Path: "/eos/dev", Kind: eos.EntryKindContainer},
		Entries: []eos.Entry{
			{Name: "example", Path: "/eos/dev/example", Kind: eos.EntryKindFile},
		},
	}
	m.nsSelected = 0
	m.nsAttrsTargetPath = "/eos/dev/example"
	m.nsAttrsLoaded = true
	m.nsAttrs = []eos.NamespaceAttr{
		{Key: "sys.acl", Value: "u:1000:rwx"},
		{Key: "user.comment", Value: "hello"},
		{Key: "user.owner", Value: "team-a"},
	}
	m.splash.active = false

	view := m.View()
	for _, needle := range []string{"Attributes", "sys.acl = u:1000:rwx", "user.comment = hello", "user.owner = team-a"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected namespace details to contain %q, got:\n%s", needle, view)
		}
	}
	if strings.Contains(view, "... more attributes") {
		t.Fatalf("expected namespace details to render all available attributes when space allows, got:\n%s", view)
	}
}

func TestDirectoryLoadedStartsNamespaceAttrLoad(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.client = &eos.Client{}
	m.activeView = viewNamespace

	updated, cmd := m.Update(directoryLoadedMsg{
		directory: eos.Directory{
			Path: "/eos/dev",
			Self: eos.Entry{Name: "dev", Path: "/eos/dev", Kind: eos.EntryKindContainer},
			Entries: []eos.Entry{
				{Name: "file-a", Path: "/eos/dev/file-a", Kind: eos.EntryKindFile},
			},
		},
	})
	m = updated.(model)

	if !m.nsAttrsLoading {
		t.Fatalf("expected namespace attrs loading after directory load")
	}
	if m.nsAttrsTargetPath != "/eos/dev/file-a" {
		t.Fatalf("expected namespace attr target path to follow selection, got %q", m.nsAttrsTargetPath)
	}
	if cmd == nil {
		t.Fatalf("expected directory load to trigger attr load command")
	}
}

func TestNamespaceAttrResponseIgnoresStaleTarget(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.nsAttrsTargetPath = "/new"
	m.nsAttrsLoading = true

	updated, _ := m.Update(namespaceAttrsLoadedMsg{
		path:  "/old",
		attrs: []eos.NamespaceAttr{{Key: "sys.old", Value: "1"}},
	})
	m = updated.(model)

	if len(m.nsAttrs) != 0 {
		t.Fatalf("expected stale namespace attrs to be ignored, got %+v", m.nsAttrs)
	}
	if !m.nsAttrsLoading {
		t.Fatalf("expected loading state to remain for current target")
	}
}

func TestNamespaceEnterOpensAttributeEditor(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.activeView = viewNamespace
	m.nsLoaded = true
	m.nsLoading = false
	m.directory = eos.Directory{
		Path: "/eos/dev",
		Self: eos.Entry{Name: "dev", Path: "/eos/dev", Kind: eos.EntryKindContainer},
		Entries: []eos.Entry{
			{Name: "file-a", Path: "/eos/dev/file-a", Kind: eos.EntryKindFile},
		},
	}
	m.nsAttrsTargetPath = "/eos/dev/file-a"
	m.nsAttrsLoaded = true
	m.nsAttrs = []eos.NamespaceAttr{
		{Key: "sys.acl", Value: "u:1000:rwx"},
		{Key: "user.comment", Value: "hello"},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)

	if !m.nsAttrEdit.active {
		t.Fatalf("expected namespace attr editor to open on enter")
	}
	if m.nsAttrEdit.stage != attrEditStageSelect {
		t.Fatalf("expected attr editor to open in key selection stage, got %d", m.nsAttrEdit.stage)
	}
	if m.nsAttrEdit.targetPath != "/eos/dev/file-a" {
		t.Fatalf("expected attr editor target path to match selection, got %q", m.nsAttrEdit.targetPath)
	}
}

func TestNamespaceAttrEditorPrefillsExistingValue(t *testing.T) {
	m := NewModel(nil, "local", "/").(model)
	m.activeView = viewNamespace
	m.nsLoaded = true
	m.nsLoading = false
	m.directory = eos.Directory{
		Path: "/eos/dev",
		Self: eos.Entry{Name: "dev", Path: "/eos/dev", Kind: eos.EntryKindContainer},
		Entries: []eos.Entry{
			{Name: "file-a", Path: "/eos/dev/file-a", Kind: eos.EntryKindFile},
		},
	}
	m.nsAttrsTargetPath = "/eos/dev/file-a"
	m.nsAttrsLoaded = true
	m.nsAttrs = []eos.NamespaceAttr{
		{Key: "sys.acl", Value: "u:1000:rwx"},
		{Key: "user.comment", Value: "hello"},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)

	if m.nsAttrEdit.stage != attrEditStageInput {
		t.Fatalf("expected enter on selected key to move to input stage, got %d", m.nsAttrEdit.stage)
	}
	if m.nsAttrEdit.input.Value() != "hello" {
		t.Fatalf("expected attr editor input to start from existing value, got %q", m.nsAttrEdit.input.Value())
	}
	if cmd == nil {
		t.Fatalf("expected attr editor to return a focus command when entering input mode")
	}
}

// ---- hotkey / legend regression tests -------------------------------------

func TestFKeyDoesNotOpenFilterInFSTView(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.activeView = viewFST
	m.fsts = []eos.FstRecord{{Host: "a", Port: 1095, Status: "online", FileSystemCount: 1}}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m = updated.(model)
	if m.popup.active {
		t.Fatalf("'f' must no longer open the filter popup in FST view; use '/' instead")
	}
}

func TestFKeyDoesNotOpenFilterInFSView(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.activeView = viewFileSystems
	m.fileSystems = []eos.FileSystemRecord{{ID: 1, Host: "h", Path: "/p", Active: "online"}}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m = updated.(model)
	if m.popup.active {
		t.Fatalf("'f' must no longer open the filter popup in FS view; use '/' instead")
	}
}

func TestEnterDoesNotSortInFSTView(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.activeView = viewFST
	m.fsts = []eos.FstRecord{{Host: "a", Port: 1095, Status: "online", FileSystemCount: 1}}
	m.fstColumnSelected = int(fstFilterHost)
	before := m.fstSort

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.fstSort != before {
		t.Fatalf("enter must not sort in FST view; sort changed from %+v to %+v", before, m.fstSort)
	}
}

func TestEnterDoesNotSortInFSView(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.activeView = viewFileSystems
	m.fileSystems = []eos.FileSystemRecord{{ID: 1, Host: "h", Path: "/p", Active: "online"}}
	m.fsColumnSelected = int(fsFilterHost)
	before := m.fsSort

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	// Enter should open the configstatus edit, not sort.
	if m.fsSort != before {
		t.Fatalf("enter must not sort in FS view; sort changed from %+v to %+v", before, m.fsSort)
	}
	if !m.fsEdit.active {
		t.Fatalf("enter should open configstatus edit popup in FS view")
	}
}

func TestLegendShowsShellAndLogsOnlyForHostViews(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 30

	hostViews := []viewID{viewMGM, viewQDB, viewFST, viewFileSystems}
	noHostViews := []viewID{viewSpaces, viewNamespaceStats, viewSpaceStatus, viewIOShaping, viewGroups}

	for _, v := range hostViews {
		m.activeView = v
		footer := m.renderFooter()
		// FS view has its own label but also contains "logs" and "shell"
		if !strings.Contains(footer, "logs") {
			t.Errorf("view %d: expected 'logs' in footer, got: %s", v, footer)
		}
		if !strings.Contains(footer, "shell") {
			t.Errorf("view %d: expected 'shell' in footer, got: %s", v, footer)
		}
		if !strings.Contains(footer, "L commands") {
			t.Errorf("view %d: expected 'L commands' in footer, got: %s", v, footer)
		}
	}

	for _, v := range noHostViews {
		m.activeView = v
		footer := m.renderFooter()
		if strings.Contains(footer, "shell") {
			t.Errorf("view %d: 'shell' should not appear in footer for non-host views, got: %s", v, footer)
		}
		if v != viewNamespace && strings.Contains(footer, " logs") {
			t.Errorf("view %d: 'logs' should not appear in footer for non-host views, got: %s", v, footer)
		}
		if !strings.Contains(footer, "L commands") {
			t.Errorf("view %d: expected 'L commands' in footer, got: %s", v, footer)
		}
	}
}

func TestLegendShowsSlashFilterNotF(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 30

	for _, v := range []viewID{viewFST, viewFileSystems, viewMGM, viewQDB} {
		m.activeView = v
		footer := m.renderFooter()
		if strings.Contains(footer, "f/") || strings.Contains(footer, "f filter") {
			t.Errorf("view %d: footer must not show 'f' as a filter hotkey, got: %s", v, footer)
		}
	}
}

func TestLegendShowsZeroToNineAndHidesHalfPageHint(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 30

	for _, v := range []viewID{viewMGM, viewFST, viewNamespace, viewIOShaping, viewGroups} {
		m.activeView = v
		footer := m.renderFooter()
		if !strings.Contains(footer, "tab/0-9") {
			t.Errorf("view %d: expected footer to show tab/0-9, got: %s", v, footer)
		}
		if strings.Contains(footer, "tab/1-0") {
			t.Errorf("view %d: footer should not show tab/1-0, got: %s", v, footer)
		}
		if strings.Contains(footer, "ctrl+d/u") {
			t.Errorf("view %d: footer should not show ctrl+d/u, got: %s", v, footer)
		}
	}
}

func TestFSConfigStatusEditOpensOnEnter(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.activeView = viewFileSystems
	m.fileSystemsLoading = false
	m.fileSystems = []eos.FileSystemRecord{
		{ID: 7, Host: "fst01", Path: "/data/01", ConfigStatus: "rw", Active: "online"},
	}
	m.fsSelected = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)

	if !m.fsEdit.active {
		t.Fatalf("expected fsEdit to be active after pressing enter in FS view")
	}
	if m.fsEdit.fsID != 7 {
		t.Errorf("expected fsEdit.fsID=7, got %d", m.fsEdit.fsID)
	}
	if m.fsEdit.current != "rw" {
		t.Errorf("expected fsEdit.current=rw, got %q", m.fsEdit.current)
	}
	// The selection should start at index 0 ("rw") since that's the current value.
	if m.fsEdit.selected != 0 {
		t.Errorf("expected fsEdit.selected=0 (rw), got %d", m.fsEdit.selected)
	}
}

func TestFSConfigStatusEditNavigation(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.activeView = viewFileSystems
	m.fileSystems = []eos.FileSystemRecord{
		{ID: 3, Host: "h", Path: "/p", ConfigStatus: "rw", Active: "online"},
	}
	m.fsSelected = 0

	// Open the popup.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if !m.fsEdit.active {
		t.Fatalf("expected fsEdit popup to open")
	}

	// Navigate down (rw → ro).
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)
	if m.fsEdit.selected != 1 {
		t.Errorf("expected selected=1 (ro) after down, got %d", m.fsEdit.selected)
	}

	// Navigate down again (ro → drain).
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)
	if m.fsEdit.selected != 2 {
		t.Errorf("expected selected=2 (drain) after second down, got %d", m.fsEdit.selected)
	}

	// Navigate down again (drain → "").
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)
	if m.fsEdit.selected != 3 {
		t.Errorf("expected selected=3 (empty) after third down, got %d", m.fsEdit.selected)
	}

	// Navigating down at end clamps.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)
	if m.fsEdit.selected != 3 {
		t.Errorf("expected clamped at 3 (last option), got %d", m.fsEdit.selected)
	}

	// Esc closes without applying.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(model)
	if m.fsEdit.active {
		t.Fatalf("expected fsEdit to close on esc")
	}
}

func TestErrorAlertDismissedByEnter(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.alert = errorAlert{active: true, message: "something went wrong"}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.alert.active {
		t.Fatalf("expected alert to be dismissed after pressing enter")
	}
}

func TestErrorAlertDismissedByEsc(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.alert = errorAlert{active: true, message: "something went wrong"}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(model)
	if m.alert.active {
		t.Fatalf("expected alert to be dismissed after pressing esc")
	}
}

func TestFSConfigStatusResultMsgShowsAlertOnError(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.fsEdit = fsConfigStatusEdit{active: true, fsID: 5}

	updated, _ := m.Update(fsConfigStatusResultMsg{err: fmt.Errorf("permission denied")})
	m = updated.(model)

	if m.fsEdit.active {
		t.Fatalf("expected fsEdit to be closed after result")
	}
	if !m.alert.active {
		t.Fatalf("expected error alert to be shown on failure")
	}
	if !strings.Contains(m.alert.message, "permission denied") {
		t.Errorf("expected alert message to contain 'permission denied', got %q", m.alert.message)
	}
}

func TestShiftLToggleShowsAndHidesCommandPanel(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	m = updated.(model)
	if !m.commandLog.active {
		t.Fatalf("expected command panel to become active")
	}
	if cmd == nil {
		t.Fatalf("expected command panel toggle to schedule a load")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	m = updated.(model)
	if m.commandLog.active {
		t.Fatalf("expected command panel to close on second Shift+L")
	}
}

func TestCommandPanelRendersRecentCommands(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 30
	m.splash.active = false
	m.commandLog.active = true
	m.commandLog.filePath = "/tmp/eos-tui.log"
	m.commandLog.lines = []string{
		"[2026-04-09 10:00:00] eos -j node ls",
		"[2026-04-09 10:00:01] ssh -o BatchMode=yes root@host 'eos -j fs ls'",
	}

	view := m.View()
	for _, needle := range []string{"Recent commands", "eos -j node ls", "/tmp/eos-tui.log"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("expected command panel view to contain %q, got:\n%s", needle, view)
		}
	}
}

func TestCommandPanelLayoutKeepsFooterAnchored(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 30
	m.activeView = viewFST
	m.commandLog.active = true
	m.commandLog.lines = []string{
		"[2026-04-09 10:00:00] eos -j node ls",
	}
	m.fstsLoading = false
	m.fsts = []eos.FstRecord{
		{Host: "fst01", Port: 1095, Status: "online", Activated: "on", FileSystemCount: 1},
	}

	view := m.View()
	assertCommandPanelAnchored(t, m, view)
}

func TestCommandPanelStaysBottomAnchoredInGroupsView(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 30
	m.activeView = viewGroups
	m.commandLog.active = true
	m.commandLog.lines = []string{
		"[2026-04-09 10:00:00] eos group ls",
	}
	m.groupsLoading = false
	m.groups = []eos.GroupRecord{
		{Name: "default.0", Status: "online", NoFS: 3},
	}

	view := m.View()
	assertCommandPanelAnchored(t, m, view)
}

func TestCommandPanelStaysBottomAnchoredWithFSConfigPopup(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 30
	m.activeView = viewFileSystems
	m.commandLog.active = true
	m.commandLog.lines = []string{
		"[2026-04-09 10:00:00] eos fs config 7 configstatus=rw",
	}
	m.fileSystemsLoading = false
	m.fileSystems = []eos.FileSystemRecord{
		{ID: 7, Host: "fst01", Port: 1095, Path: "/data/01", ConfigStatus: "rw", Active: "online"},
	}
	m.fsSelected = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if !m.fsEdit.active {
		t.Fatalf("expected fs config popup to open")
	}

	view := m.View()
	assertCommandPanelAnchored(t, m, view)
	if !strings.Contains(view, "/data/01") {
		t.Fatalf("expected filesystem mount path to remain visible with popup open, got:\n%s", view)
	}
}

func TestGroupsViewDoesNotInsertBlankLineBeforeCommandPanel(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 30
	m.activeView = viewGroups
	m.commandLog.active = true
	m.commandLog.lines = []string{
		"[2026-04-09 10:00:00] eos group ls",
	}
	m.groupsLoading = false
	m.groups = []eos.GroupRecord{
		{Name: "default.0", Status: "online", NoFS: 3},
	}

	view := m.View()
	assertCommandPanelAnchored(t, m, view)

	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	headerHeight := lipgloss.Height(m.renderHeader())
	footerHeight := lipgloss.Height(m.renderFooter())
	middleHeight := max(0, m.height-headerHeight-footerHeight)
	availableHeight := max(4, middleHeight-2)
	_, commandHeight := m.splitMainAndCommandHeights(availableHeight)
	commandTitleLine := headerHeight + (middleHeight - commandHeight) + 1
	if commandTitleLine == 0 {
		t.Fatalf("unexpected command title line")
	}
	if commandTitleLine < 2 {
		t.Fatalf("unexpected command title line %d", commandTitleLine)
	}
	if prev := strings.TrimSpace(lines[commandTitleLine-1]); prev == "" {
		t.Fatalf("expected no blank spacer line before command panel, got:\n%s", view)
	}
	if prevPrev := strings.TrimSpace(lines[commandTitleLine-2]); prevPrev == "" {
		t.Fatalf("expected no extra blank line between group details and command panel, got:\n%s", view)
	}
	if !strings.Contains(view, "Free") {
		t.Fatalf("expected selected group details to include the Free row, got:\n%s", view)
	}
}

func TestCommandLogTickDoesNotReenterLoadingAfterInitialData(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.commandLog.active = true
	m.commandLog.loading = false
	m.commandLog.lines = []string{"[2026-04-09 12:22:08] eos -j fs ls"}

	updated, cmd := m.Update(commandLogTickMsg{})
	m = updated.(model)

	if m.commandLog.loading {
		t.Fatalf("expected command log refresh to keep existing content visible without setting loading=true")
	}
	if cmd == nil {
		t.Fatalf("expected refresh tick to schedule the next command log load")
	}
}
