package ui

import (
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lobis/eos-tui/eos"
)

func lineCount(s string) int {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func TestNewModelRendersMenuWithoutWindowSize(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)

	view := m.View()
	for _, needle := range []string{"EOS TUI", "1 MGM", "2 QDB", "3 FST", "4 FS", "5 Namespace"} {
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
	if !strings.Contains(view, "EOS TUI") {
		t.Fatalf("expected rendered view to still contain menu header, got:\n%s", view)
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
				HostPort:        "host:1095",
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
	for _, needle := range []string{"host:1095", "Cluster Summary", "Selected Node", "online"} {
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
		HostPort:        "lobisapa-dev.cern.ch:1095",
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

func TestVisibleFSTsFilterByStatus(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.fsts = []eos.FstRecord{
		{HostPort: "b:1095", Status: "offline", FileSystemCount: 1},
		{HostPort: "a:1095", Status: "online", FileSystemCount: 5},
		{HostPort: "c:1095", Status: "online", FileSystemCount: 3},
	}
	// fstFilterStatus corresponds to filter column 3; we set both .column and
	// .filters so visibleFSTs() applies the filter.
	m.fstFilter.column = int(fstFilterStatus)
	m.fstFilter.filters[int(fstFilterStatus)] = "online"

	fsts := m.visibleFSTs()
	if len(fsts) != 2 {
		t.Fatalf("expected 2 filtered fsts, got %d", len(fsts))
	}
	if fsts[0].HostPort != "a:1095" || fsts[1].HostPort != "c:1095" {
		t.Fatalf("unexpected filtered order: %#v", fsts)
	}
}

func TestVisibleFSTsSortByFileSystemsDesc(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.fsts = []eos.FstRecord{
		{HostPort: "b:1095", Status: "online", FileSystemCount: 1},
		{HostPort: "a:1095", Status: "online", FileSystemCount: 5},
		{HostPort: "c:1095", Status: "online", FileSystemCount: 3},
	}
	m.fstSort.column = int(fstSortFileSystems)
	m.fstSort.desc = true

	fsts := m.visibleFSTs()
	if got := []string{fsts[0].HostPort, fsts[1].HostPort, fsts[2].HostPort}; strings.Join(got, ",") != "a:1095,c:1095,b:1095" {
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

func TestFilterPopupAppliesToFSTs(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.activeView = viewFST
	m.fsts = []eos.FstRecord{
		{HostPort: "alpha:1095", Status: "online", FileSystemCount: 2},
		{HostPort: "beta:1095", Status: "offline", FileSystemCount: 1},
	}
	m.fstColumnSelected = int(fstFilterHostPort)

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
	if len(fsts) != 1 || fsts[0].HostPort != "beta:1095" {
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
			{HostPort: "fast-node:1095", Status: "online", Activated: "on", Geotag: "local", FileSystemCount: 5},
		},
	})
	m = updated.(model)

	view := m.View()
	if !strings.Contains(view, "fast-node:1095") {
		t.Fatalf("expected node table to render before stats load, got:\n%s", view)
	}
	if !strings.Contains(view, "Loading cluster summary...") {
		t.Fatalf("expected summary area to show incremental loading state, got:\n%s", view)
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

	// Key '5' switches to the Namespace view (tab 5 in the new layout).
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
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
		{HostPort: "a:1095", Status: "online", FileSystemCount: 1},
		{HostPort: "b:1095", Status: "offline", FileSystemCount: 1},
	}
	m.fstColumnSelected = int(fstFilterStatus)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
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
	// Display column 0 = "hostport".  The fstFilter/fstSort enums have an extra
	// "type" entry at 0 that doesn't appear in the display, so the display column
	// index (0) and the semantic enum value (fstFilterHostPort = 1) are different.
	// We test with raw display-column index 0 which maps to "hostport".
	m.fstColumnSelected = 0
	m.fstSort = sortState{column: 0} // column-0 indicator appears on "hostport"
	m.fsts = []eos.FstRecord{{HostPort: "a:1095", FileSystemCount: 1}}

	view := m.renderNodesList(m.contentWidth(), 10)
	if !strings.Contains(view, "[hostport") || !strings.Contains(view, "hostport↑") {
		t.Fatalf("expected header to show selected sorted column, got:\n%s", view)
	}
}

func TestFilterPopupCanBeCancelled(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.activeView = viewFST
	m.fsts = []eos.FstRecord{{HostPort: "alpha:1095", FileSystemCount: 1}}
	m.fstColumnSelected = int(fstFilterHostPort)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m = updated.(model)
	if !m.popup.active {
		t.Fatalf("expected popup to open")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(model)
	if m.popup.active {
		t.Fatalf("expected popup to close on escape")
	}
	if m.fstFilter.filters[int(fstFilterHostPort)] != "" {
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
	m.fsts = []eos.FstRecord{{HostPort: "fst:1095", Status: "online", Activated: "on", FileSystemCount: 1}}
	m.fstsLoading = false
	m.mgms = []eos.MgmRecord{{HostPort: "mgm:1094", QDBHostPort: "mgm:7777", Role: "leader", Status: "online", EOSVersion: "5.x"}}
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

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'6'}})
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

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'7'}})
	m = updated.(model)

	if !m.nsStatsLoading {
		t.Fatalf("expected nsStatsLoading=true after switching to namespace stats view")
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
	m.namespaceStats = eos.NamespaceStats{
		TotalFiles:       78,
		TotalDirectories: 19,
		CurrentFID:       7661,
		CurrentCID:       1234,
		MasterHost:       "mgm01:1094",
	}

	view := m.View()
	for _, needle := range []string{"Namespace Statistics", "78", "19"} {
		if !strings.Contains(view, needle) {
			t.Errorf("expected namespace stats view to contain %q, got:\n%s", needle, view)
		}
	}
}

func TestSpacesViewShowsLoadingState(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 30
	m.activeView = viewSpaces
	m.spacesLoading = true

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
		{HostPort: "fst01.cern.ch:1095", Status: "online", Type: "fst"},
		{HostPort: "fst02.cern.ch:1095", Status: "online", Type: "fst"},
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
		{HostPort: "mgm01.cern.ch:1094", Role: "leader"},
		{HostPort: "mgm02.cern.ch:1094", Role: "follower"},
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
		{QDBHostPort: "qdb01.cern.ch:7777", Role: "leader"},
		{QDBHostPort: "qdb02.cern.ch:7777", Role: "follower"},
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
		{HostPort: "mgm01:1094", Role: "leader"},
		{HostPort: "mgm02:1094", Role: "follower"},
		{HostPort: "mgm03:1094", Role: "follower"},
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
		{HostPort: "mgm01:1094"},
		{HostPort: "mgm02:1094"},
		{HostPort: "mgm03:1094"},
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
		{QDBHostPort: "qdb01:7777", Role: "leader"},
		{QDBHostPort: "qdb02:7777", Role: "follower"},
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
		{HostPort: "mgm01.cern.ch:1094", Role: "leader", Status: "online"},
		{HostPort: "mgm02.cern.ch:1094", Role: "follower", Status: "online"},
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
		{QDBHostPort: "qdb01.cern.ch:7777", Role: "leader", Status: "online", EOSVersion: "5.3.29"},
		{QDBHostPort: "qdb02.cern.ch:7777", Role: "follower", Status: "online", EOSVersion: "5.3.29"},
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
		{HostPort: "mgm01.cern.ch:1094"},
		{HostPort: "mgm02.cern.ch:1094"},
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
		HostPort:        "fst01.cern.ch:1095",
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
		{fstFilterHostPort, "hostport", node.HostPort},
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
		{HostPort: "alpha.cern.ch:1095", Type: "fst", FileSystemCount: 1},
		{HostPort: "beta.cern.ch:1095", Type: "fst", FileSystemCount: 1},
		{HostPort: "gamma.cern.ch:1095", Type: "fst", FileSystemCount: 1},
	}

	// Filter hostport column for "alpha".
	m.fstFilter.filters = map[int]string{int(fstFilterHostPort): "alpha"}

	visible := m.visibleFSTs()
	if len(visible) != 1 {
		t.Fatalf("expected 1 visible FST after hostport filter, got %d", len(visible))
	}
	if visible[0].HostPort != "alpha.cern.ch:1095" {
		t.Errorf("expected alpha.cern.ch:1095, got %q", visible[0].HostPort)
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
