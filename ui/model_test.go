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
	for _, needle := range []string{"EOS TUI", "1 Nodes", "2 Filesystems", "3 Namespace", "Loading EOS state..."} {
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
		nodes: []eos.FstRecord{
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

	view := m.View()
	for _, needle := range []string{"host:1095", "Cluster Summary", "Selected Node", "online", "Connected to local eos cli"} {
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
	m.nodesLoading = false
	m.nodeStats = eos.NodeStats{
		State:       "OK",
		ThreadCount: 489,
		FileCount:   78,
		DirCount:    19,
		FileDescs:   553,
	}
	m.nodes = []eos.FstRecord{{
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

func TestVisibleNodesFilterByStatus(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.nodes = []eos.FstRecord{
		{HostPort: "b:1095", Status: "offline", FileSystemCount: 1},
		{HostPort: "a:1095", Status: "online", FileSystemCount: 5},
		{HostPort: "c:1095", Status: "online", FileSystemCount: 3},
	}
	m.nodeFilter.column = int(nodeFilterStatus)
	m.nodeFilter.filters[int(nodeFilterStatus)] = "online"

	nodes := m.visibleNodes()
	if len(nodes) != 2 {
		t.Fatalf("expected 2 filtered nodes, got %d", len(nodes))
	}
	if nodes[0].HostPort != "a:1095" || nodes[1].HostPort != "c:1095" {
		t.Fatalf("unexpected filtered order: %#v", nodes)
	}
}

func TestVisibleNodesSortByFileSystemsDesc(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.nodes = []eos.FstRecord{
		{HostPort: "b:1095", Status: "online", FileSystemCount: 1},
		{HostPort: "a:1095", Status: "online", FileSystemCount: 5},
		{HostPort: "c:1095", Status: "online", FileSystemCount: 3},
	}
	m.nodeSort.column = int(nodeSortFileSystems)
	m.nodeSort.desc = true

	nodes := m.visibleNodes()
	if got := []string{nodes[0].HostPort, nodes[1].HostPort, nodes[2].HostPort}; strings.Join(got, ",") != "a:1095,c:1095,b:1095" {
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

func TestFilterPopupAppliesToNodes(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.nodes = []eos.FstRecord{
		{HostPort: "alpha:1095", Status: "online"},
		{HostPort: "beta:1095", Status: "offline"},
	}
	m.nodeColumnSelected = int(nodeFilterHostPort)

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

	nodes := m.visibleNodes()
	if len(nodes) != 1 || nodes[0].HostPort != "beta:1095" {
		t.Fatalf("expected filter input to keep beta only, got %#v", nodes)
	}
}

func TestNodesRenderBeforeStatsArrive(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.width = 120
	m.height = 24

	updated, _ := m.Update(nodesLoadedMsg{
		nodes: []eos.FstRecord{
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

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = updated.(model)

	if !m.nsLoading {
		t.Fatalf("expected namespace load to start when switching to namespace view")
	}
	if cmd == nil {
		t.Fatalf("expected namespace switch to return a load command")
	}
}

func TestNodeSortCyclesOnSelectedColumn(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.nodeColumnSelected = int(nodeFilterNoFS)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(model)
	if m.nodeSort.column != int(nodeSortNoFS) || m.nodeSort.desc {
		t.Fatalf("expected first sort press to set ascending nofs sort, got %+v", m.nodeSort)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(model)
	if m.nodeSort.column != int(nodeSortNoFS) || !m.nodeSort.desc {
		t.Fatalf("expected second sort press to set descending nofs sort, got %+v", m.nodeSort)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(model)
	if m.nodeSort.column != int(nodeSortNone) {
		t.Fatalf("expected third sort press to clear sort, got %+v", m.nodeSort)
	}
}

func TestNodeEnumFilterCyclesOnSelectedColumn(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.nodes = []eos.FstRecord{
		{HostPort: "a:1095", Status: "online"},
		{HostPort: "b:1095", Status: "offline"},
	}
	m.nodeColumnSelected = int(nodeFilterStatus)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m = updated.(model)
	if !m.popup.active {
		t.Fatalf("expected popup to open for enum filter")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o', 'f', 'f'}})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.nodeFilter.column != int(nodeFilterStatus) || m.nodeFilter.filters[int(nodeFilterStatus)] != "offline" {
		t.Fatalf("expected enum popup selection to apply offline filter, got %+v", m.nodeFilter)
	}
}

func TestHeaderShowsSelectedAndSortedColumn(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.nodesLoading = false
	m.nodeColumnSelected = int(nodeFilterHostPort)
	m.nodeSort = sortState{column: int(nodeSortHostPort)}
	m.nodes = []eos.FstRecord{{HostPort: "a:1095"}}

	view := m.renderNodesList(m.contentWidth(), 10)
	if !strings.Contains(view, "[hostport") || !strings.Contains(view, "hostport↑") {
		t.Fatalf("expected header to show selected sorted column, got:\n%s", view)
	}
}

func TestFilterPopupCanBeCancelled(t *testing.T) {
	m := NewModel(nil, "local eos cli", "/").(model)
	m.nodes = []eos.FstRecord{{HostPort: "alpha:1095"}}
	m.nodeColumnSelected = int(nodeFilterHostPort)

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
	if m.nodeFilter.filters[int(nodeFilterHostPort)] != "" {
		t.Fatalf("expected filter to remain unchanged after cancel, got %+v", m.nodeFilter)
	}
}

func padIndex(i int) string {
	if i < 10 {
		return "0" + strconv.Itoa(i)
	}
	return strconv.Itoa(i)
}
