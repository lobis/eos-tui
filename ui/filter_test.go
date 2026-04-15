package ui

import (
	"testing"

	"github.com/lobis/eos-tui/eos"
)

func newTestModel(t *testing.T) model {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	return NewModel(nil, "test", "/").(model)
}

// ---------------------------------------------------------------------------
// visibleFileSystems
// ---------------------------------------------------------------------------

func TestVisibleFileSystemsFiltersAndSorts(t *testing.T) {
	m := newTestModel(t)
	m.fileSystems = []eos.FileSystemRecord{
		{Host: "alpha", ID: 3, Path: "/a"},
		{Host: "beta", ID: 1, Path: "/b"},
		{Host: "alpha", ID: 2, Path: "/c"},
	}

	// No filter, no sort → original order preserved.
	got := m.visibleFileSystems()
	if len(got) != 3 {
		t.Fatalf("expected 3, got %d", len(got))
	}

	// Filter by host containing "alpha".
	m.fsFilter = filterState{column: int(fsFilterHost), filters: map[int]string{int(fsFilterHost): "alpha"}}
	got = m.visibleFileSystems()
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	for _, fs := range got {
		if fs.Host != "alpha" {
			t.Fatalf("expected host alpha, got %s", fs.Host)
		}
	}

	// Sort ascending by ID.
	m.fsFilter = filterState{filters: map[int]string{}}
	m.fsSort = sortState{column: int(fsSortID)}
	got = m.visibleFileSystems()
	if got[0].ID != 1 || got[1].ID != 2 || got[2].ID != 3 {
		t.Fatalf("expected IDs [1,2,3], got [%d,%d,%d]", got[0].ID, got[1].ID, got[2].ID)
	}

	// Sort descending.
	m.fsSort = sortState{column: int(fsSortID), desc: true}
	got = m.visibleFileSystems()
	if got[0].ID != 3 || got[1].ID != 2 || got[2].ID != 1 {
		t.Fatalf("expected IDs [3,2,1], got [%d,%d,%d]", got[0].ID, got[1].ID, got[2].ID)
	}

	// Filter + sort combined.
	m.fsFilter = filterState{filters: map[int]string{int(fsFilterHost): "alpha"}}
	m.fsSort = sortState{column: int(fsSortID)}
	got = m.visibleFileSystems()
	if len(got) != 2 || got[0].ID != 2 || got[1].ID != 3 {
		t.Fatalf("expected filtered+sorted [2,3], got %v", got)
	}
}

// ---------------------------------------------------------------------------
// visibleGroups
// ---------------------------------------------------------------------------

func TestVisibleGroupsFiltersAndSorts(t *testing.T) {
	m := newTestModel(t)
	m.groups = []eos.GroupRecord{
		{Name: "default.1", Status: "on", NoFS: 5},
		{Name: "default.2", Status: "off", NoFS: 3},
		{Name: "spare.1", Status: "on", NoFS: 10},
	}

	// Unfiltered.
	got := m.visibleGroups()
	if len(got) != 3 {
		t.Fatalf("expected 3, got %d", len(got))
	}

	// Filter by name "default".
	m.groupFilter = filterState{filters: map[int]string{int(groupFilterName): "default"}}
	got = m.visibleGroups()
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}

	// Sort by NoFS ascending.
	m.groupFilter = filterState{filters: map[int]string{}}
	m.groupSort = sortState{column: int(groupSortNoFS)}
	got = m.visibleGroups()
	if got[0].NoFS != 3 || got[1].NoFS != 5 || got[2].NoFS != 10 {
		t.Fatalf("expected NoFS [3,5,10], got [%d,%d,%d]", got[0].NoFS, got[1].NoFS, got[2].NoFS)
	}

	// Sort desc.
	m.groupSort = sortState{column: int(groupSortNoFS), desc: true}
	got = m.visibleGroups()
	if got[0].NoFS != 10 {
		t.Fatalf("expected first NoFS=10, got %d", got[0].NoFS)
	}
}

// ---------------------------------------------------------------------------
// visibleSpaces
// ---------------------------------------------------------------------------

func TestVisibleSpacesFiltersAndSorts(t *testing.T) {
	m := newTestModel(t)
	m.spaces = []eos.SpaceRecord{
		{Name: "default", Status: "on", Groups: 5, UsedBytes: 80, CapacityBytes: 100},
		{Name: "scratch", Status: "off", Groups: 2, UsedBytes: 10, CapacityBytes: 100},
		{Name: "default-drain", Status: "drain", Groups: 9, UsedBytes: 40, CapacityBytes: 100},
	}

	got := m.visibleSpaces()
	if len(got) != 3 {
		t.Fatalf("expected 3, got %d", len(got))
	}

	m.spaceFilter = filterState{filters: map[int]string{int(spaceFilterName): "default"}}
	got = m.visibleSpaces()
	if len(got) != 2 {
		t.Fatalf("expected 2 filtered spaces, got %d", len(got))
	}

	m.spaceFilter = filterState{filters: map[int]string{}}
	m.spaceSort = sortState{column: int(spaceSortGroups)}
	got = m.visibleSpaces()
	if got[0].Groups != 2 || got[1].Groups != 5 || got[2].Groups != 9 {
		t.Fatalf("expected groups [2,5,9], got [%d,%d,%d]", got[0].Groups, got[1].Groups, got[2].Groups)
	}

	m.spaceSort = sortState{column: int(spaceSortUsage), desc: true}
	got = m.visibleSpaces()
	if got[0].Name != "default" || got[1].Name != "default-drain" || got[2].Name != "scratch" {
		t.Fatalf("unexpected usage sort order: [%s,%s,%s]", got[0].Name, got[1].Name, got[2].Name)
	}
}

// ---------------------------------------------------------------------------
// matchesFileSystemFilters
// ---------------------------------------------------------------------------

func TestMatchesFileSystemFilters(t *testing.T) {
	m := newTestModel(t)
	fs := eos.FileSystemRecord{Host: "node01", Port: 1095, SchedGroup: "default.0"}

	// No filters → always match.
	m.fsFilter = filterState{filters: map[int]string{}}
	if !m.matchesFileSystemFilters(fs) {
		t.Fatal("expected match with no filters")
	}

	// Single filter match.
	m.fsFilter.filters[int(fsFilterHost)] = "node"
	if !m.matchesFileSystemFilters(fs) {
		t.Fatal("expected match on host substring")
	}

	// Single filter mismatch.
	m.fsFilter.filters[int(fsFilterHost)] = "xyz"
	if m.matchesFileSystemFilters(fs) {
		t.Fatal("expected no match")
	}

	// Multiple filters – all must match.
	m.fsFilter.filters = map[int]string{
		int(fsFilterHost):  "node",
		int(fsFilterGroup): "default",
	}
	if !m.matchesFileSystemFilters(fs) {
		t.Fatal("expected match with two filters")
	}

	// One of multiple fails.
	m.fsFilter.filters[int(fsFilterGroup)] = "spare"
	if m.matchesFileSystemFilters(fs) {
		t.Fatal("expected no match when one filter fails")
	}

	// Case insensitive.
	m.fsFilter.filters = map[int]string{int(fsFilterHost): "NODE"}
	if !m.matchesFileSystemFilters(fs) {
		t.Fatal("expected case-insensitive match")
	}
}

// ---------------------------------------------------------------------------
// matchesFileSystemFiltersExcept
// ---------------------------------------------------------------------------

func TestMatchesFileSystemFiltersExcept(t *testing.T) {
	m := newTestModel(t)
	fs := eos.FileSystemRecord{Host: "node01", SchedGroup: "default.0"}

	m.fsFilter = filterState{filters: map[int]string{
		int(fsFilterHost):  "node",
		int(fsFilterGroup): "spare", // does NOT match
	}}

	// Normal match should fail.
	if m.matchesFileSystemFilters(fs) {
		t.Fatal("full match should fail")
	}

	// Excluding the group column should pass.
	if !m.matchesFileSystemFiltersExcept(fs, int(fsFilterGroup)) {
		t.Fatal("expected match when excluding group column")
	}

	// Excluding host (but group still fails) should fail.
	if m.matchesFileSystemFiltersExcept(fs, int(fsFilterHost)) {
		t.Fatal("expected no match when only host excluded")
	}
}

// ---------------------------------------------------------------------------
// matchesGroupFilters
// ---------------------------------------------------------------------------

func TestMatchesGroupFilters(t *testing.T) {
	m := newTestModel(t)
	g := eos.GroupRecord{Name: "default.0", Status: "on", NoFS: 5}

	m.groupFilter = filterState{filters: map[int]string{}}
	if !m.matchesGroupFilters(g) {
		t.Fatal("expected match with no filters")
	}

	m.groupFilter.filters[int(groupFilterName)] = "default"
	if !m.matchesGroupFilters(g) {
		t.Fatal("expected match")
	}

	m.groupFilter.filters[int(groupFilterStatus)] = "off"
	if m.matchesGroupFilters(g) {
		t.Fatal("expected no match")
	}
}

func TestMatchesSpaceFiltersExcept(t *testing.T) {
	m := newTestModel(t)
	s := eos.SpaceRecord{Name: "default", Status: "on", Groups: 5}

	m.spaceFilter = filterState{filters: map[int]string{
		int(spaceFilterName):   "default",
		int(spaceFilterStatus): "off",
	}}

	if m.matchesSpaceFilters(s) {
		t.Fatal("full match should fail")
	}
	if !m.matchesSpaceFiltersExcept(s, int(spaceFilterStatus)) {
		t.Fatal("expected match when excluding status column")
	}
	if m.matchesSpaceFiltersExcept(s, int(spaceFilterName)) {
		t.Fatal("expected mismatch when excluding only name")
	}
}

// ---------------------------------------------------------------------------
// fsFilterValueForColumn
// ---------------------------------------------------------------------------

func TestFsFilterValueForColumn(t *testing.T) {
	m := newTestModel(t)
	fs := eos.FileSystemRecord{
		Host:          "node01",
		Port:          1095,
		ID:            42,
		Path:          "/data",
		SchedGroup:    "default.0",
		Geotag:        "site::rack",
		Boot:          "booted",
		ConfigStatus:  "rw",
		DrainStatus:   "nodrain",
		Active:        "online",
		Health:        "healthy",
		UsedBytes:     50,
		CapacityBytes: 100,
	}

	cases := []struct {
		col    int
		expect string
	}{
		{int(fsFilterHost), "node01"},
		{int(fsFilterPort), "1095"},
		{int(fsFilterID), "42"},
		{int(fsFilterPath), "/data"},
		{int(fsFilterGroup), "default.0"},
		{int(fsFilterGeotag), "site::rack"},
		{int(fsFilterBoot), "booted"},
		{int(fsFilterConfigStatus), "rw"},
		{int(fsFilterDrain), "nodrain"},
		{int(fsFilterStatus), "online"},
		{int(fsFilterHealth), "healthy"},
		{int(fsFilterUsage), "50.00"},
		{999, "node01"}, // default falls back to Host
	}
	for _, tc := range cases {
		got := m.fsFilterValueForColumn(fs, tc.col)
		if got != tc.expect {
			t.Errorf("column %d: expected %q, got %q", tc.col, tc.expect, got)
		}
	}
}

// ---------------------------------------------------------------------------
// groupFilterValueForColumn
// ---------------------------------------------------------------------------

func TestGroupFilterValueForColumn(t *testing.T) {
	m := newTestModel(t)
	g := eos.GroupRecord{
		Name:          "default.0",
		Status:        "on",
		NoFS:          5,
		CapacityBytes: 1024 * 1024 * 1024, // 1 GiB
		UsedBytes:     512 * 1024 * 1024,  // 512 MiB
		FreeBytes:     512 * 1024 * 1024,
		NumFiles:      1000,
	}

	cases := []struct {
		col    int
		expect string
	}{
		{int(groupFilterName), "default.0"},
		{int(groupFilterStatus), "on"},
		{int(groupFilterNoFS), "5"},
		{int(groupFilterCapacity), humanBytes(g.CapacityBytes)},
		{int(groupFilterUsed), humanBytes(g.UsedBytes)},
		{int(groupFilterFree), humanBytes(g.FreeBytes)},
		{int(groupFilterFiles), "1000"},
		{999, "default.0"}, // default
	}
	for _, tc := range cases {
		got := m.groupFilterValueForColumn(g, tc.col)
		if got != tc.expect {
			t.Errorf("column %d: expected %q, got %q", tc.col, tc.expect, got)
		}
	}
}

func TestSpaceFilterValueForColumn(t *testing.T) {
	m := newTestModel(t)
	s := eos.SpaceRecord{
		Name:          "default",
		Type:          "space",
		Status:        "on",
		Groups:        5,
		NumFiles:      1000,
		NumContainers: 42,
		UsedBytes:     25,
		CapacityBytes: 100,
	}

	cases := []struct {
		col    int
		expect string
	}{
		{int(spaceFilterName), "default"},
		{int(spaceFilterType), "space"},
		{int(spaceFilterStatus), "on"},
		{int(spaceFilterGroups), "5"},
		{int(spaceFilterFiles), "1000"},
		{int(spaceFilterDirs), "42"},
		{int(spaceFilterUsage), "25.00"},
		{999, "default"},
	}
	for _, tc := range cases {
		if got := m.spaceFilterValueForColumn(s, tc.col); got != tc.expect {
			t.Errorf("column %d: expected %q, got %q", tc.col, tc.expect, got)
		}
	}
}

// ---------------------------------------------------------------------------
// lessNode
// ---------------------------------------------------------------------------

func TestLessNodeAllColumns(t *testing.T) {
	m := newTestModel(t)
	a := eos.FstRecord{
		Host: "aaa", Port: 1, Geotag: "a", Status: "booted",
		Activated: "false", HeartbeatDelta: 1, FileSystemCount: 2,
		EOSVersion: "4.0", Type: "fst",
	}
	b := eos.FstRecord{
		Host: "bbb", Port: 2, Geotag: "b", Status: "online",
		Activated: "true", HeartbeatDelta: 5, FileSystemCount: 8,
		EOSVersion: "5.0", Type: "gateway",
	}

	columns := []fstSortColumn{
		fstSortHost, fstSortPort, fstSortGeotag, fstSortStatus,
		fstSortActivated, fstSortHeartbeat, fstSortNoFS,
		fstSortEOSVersion, fstSortType,
	}

	for _, col := range columns {
		m.fstSort = sortState{column: int(col)}
		if !m.lessNode(a, b) {
			t.Errorf("col %d asc: expected a < b", col)
		}
		if m.lessNode(b, a) {
			t.Errorf("col %d asc: expected b >= a", col)
		}
		// Descending.
		m.fstSort.desc = true
		if !m.lessNode(b, a) {
			t.Errorf("col %d desc: expected b < a", col)
		}
	}

	// Default (unknown column) sorts by host.
	m.fstSort = sortState{column: 999}
	if !m.lessNode(a, b) {
		t.Error("default: expected a < b by host")
	}

	// Tie-breaking: same host, different port.
	c := a
	c.Port = 10
	m.fstSort = sortState{column: int(fstSortHost)}
	if m.lessNode(c, a) {
		t.Error("tie-break: expected a (port 1) before c (port 10)")
	}
}

// ---------------------------------------------------------------------------
// lessFileSystem
// ---------------------------------------------------------------------------

func TestLessFileSystemAllColumns(t *testing.T) {
	m := newTestModel(t)
	a := eos.FileSystemRecord{
		Host: "a", Port: 1, ID: 1, Path: "/a", SchedGroup: "a",
		Geotag: "a", Boot: "a", ConfigStatus: "a", DrainStatus: "a",
		Active: "a", Health: "a",
		UsedBytes: 25, CapacityBytes: 100,
	}
	b := eos.FileSystemRecord{
		Host: "b", Port: 2, ID: 2, Path: "/b", SchedGroup: "b",
		Geotag: "b", Boot: "b", ConfigStatus: "b", DrainStatus: "b",
		Active: "b", Health: "b",
		UsedBytes: 75, CapacityBytes: 100,
	}

	columns := []fsSortColumn{
		fsSortHost, fsSortPort, fsSortID, fsSortPath, fsSortGroup,
		fsSortGeotag, fsSortBoot, fsSortConfigStatus, fsSortDrain,
		fsSortUsed, fsSortStatus, fsSortHealth,
	}

	for _, col := range columns {
		m.fsSort = sortState{column: int(col)}
		if !m.lessFileSystem(a, b) {
			t.Errorf("col %d asc: expected a < b", col)
		}
		m.fsSort.desc = true
		if !m.lessFileSystem(b, a) {
			t.Errorf("col %d desc: expected b < a", col)
		}
	}

	// Default (unknown column) sorts by ID.
	m.fsSort = sortState{column: 999}
	if !m.lessFileSystem(a, b) {
		t.Error("default: expected a < b by ID")
	}
}

// ---------------------------------------------------------------------------
// lessGroup
// ---------------------------------------------------------------------------

func TestLessGroupAllColumns(t *testing.T) {
	m := newTestModel(t)
	a := eos.GroupRecord{Name: "a", Status: "a", NoFS: 1, CapacityBytes: 100, UsedBytes: 10, FreeBytes: 90, NumFiles: 1}
	b := eos.GroupRecord{Name: "b", Status: "b", NoFS: 5, CapacityBytes: 200, UsedBytes: 50, FreeBytes: 150, NumFiles: 10}

	columns := []groupSortColumn{
		groupSortName, groupSortStatus, groupSortNoFS,
		groupSortCapacity, groupSortUsed, groupSortFree, groupSortFiles,
	}

	for _, col := range columns {
		m.groupSort = sortState{column: int(col)}
		if !m.lessGroup(a, b) {
			t.Errorf("col %d asc: expected a < b", col)
		}
		m.groupSort.desc = true
		if !m.lessGroup(b, a) {
			t.Errorf("col %d desc: expected b < a", col)
		}
	}

	// Default falls back to name.
	m.groupSort = sortState{column: 999}
	if !m.lessGroup(a, b) {
		t.Error("default: expected a < b by name")
	}
}

func TestLessSpaceAllColumns(t *testing.T) {
	m := newTestModel(t)
	a := eos.SpaceRecord{Name: "a", Type: "a", Status: "a", Groups: 1, NumFiles: 10, NumContainers: 2, UsedBytes: 10, CapacityBytes: 100}
	b := eos.SpaceRecord{Name: "b", Type: "b", Status: "b", Groups: 5, NumFiles: 20, NumContainers: 4, UsedBytes: 50, CapacityBytes: 100}

	columns := []spaceSortColumn{
		spaceSortName, spaceSortType, spaceSortStatus,
		spaceSortGroups, spaceSortFiles, spaceSortDirs, spaceSortUsage,
	}

	for _, col := range columns {
		m.spaceSort = sortState{column: int(col)}
		if !m.lessSpace(a, b) {
			t.Errorf("col %d asc: expected a < b", col)
		}
		m.spaceSort.desc = true
		if !m.lessSpace(b, a) {
			t.Errorf("col %d desc: expected b < a", col)
		}
	}

	m.spaceSort = sortState{column: 999}
	if !m.lessSpace(a, b) {
		t.Error("default: expected a < b by name")
	}
}

// ---------------------------------------------------------------------------
// equivalentNodeSortValue
// ---------------------------------------------------------------------------

func TestEquivalentNodeSortValue(t *testing.T) {
	a := eos.FstRecord{
		Host: "h1", Type: "fst", Status: "online", Geotag: "g1",
		Activated: "true", FileSystemCount: 2, HeartbeatDelta: 5, EOSVersion: "4.0",
	}
	same := a
	diff := eos.FstRecord{
		Host: "h2", Type: "gateway", Status: "offline", Geotag: "g2",
		Activated: "false", FileSystemCount: 9, HeartbeatDelta: 99, EOSVersion: "5.0",
	}

	columns := []int{
		int(fstSortHost), int(fstSortType), int(fstSortStatus),
		int(fstSortGeotag), int(fstSortActivated), int(fstSortNoFS),
		int(fstSortHeartbeat), int(fstSortEOSVersion),
	}

	for _, col := range columns {
		if !equivalentNodeSortValue(col, a, same) {
			t.Errorf("col %d: expected equivalent", col)
		}
		if equivalentNodeSortValue(col, a, diff) {
			t.Errorf("col %d: expected not equivalent", col)
		}
	}

	// Default (unknown column) compares Host.
	if !equivalentNodeSortValue(999, a, same) {
		t.Error("default: expected equivalent by host")
	}
}

// ---------------------------------------------------------------------------
// equivalentFileSystemSortValue
// ---------------------------------------------------------------------------

func TestEquivalentFileSystemSortValue(t *testing.T) {
	a := eos.FileSystemRecord{
		Host: "h", Port: 1, ID: 1, Path: "/a", SchedGroup: "g",
		Geotag: "geo", Boot: "b", ConfigStatus: "rw", DrainStatus: "nodrain",
		Active: "online", Health: "ok",
		UsedBytes: 50, CapacityBytes: 100,
	}
	same := a
	diff := eos.FileSystemRecord{
		Host: "x", Port: 9, ID: 9, Path: "/z", SchedGroup: "s",
		Geotag: "other", Boot: "x", ConfigStatus: "ro", DrainStatus: "drain",
		Active: "offline", Health: "bad",
		UsedBytes: 99, CapacityBytes: 100,
	}

	columns := []int{
		int(fsSortHost), int(fsSortPort), int(fsSortID), int(fsSortPath),
		int(fsSortGroup), int(fsSortGeotag), int(fsSortBoot),
		int(fsSortConfigStatus), int(fsSortDrain), int(fsSortUsed),
		int(fsSortStatus), int(fsSortHealth),
	}

	for _, col := range columns {
		if !equivalentFileSystemSortValue(col, a, same) {
			t.Errorf("col %d: expected equivalent", col)
		}
		if equivalentFileSystemSortValue(col, a, diff) {
			t.Errorf("col %d: expected not equivalent", col)
		}
	}

	// Default compares ID.
	if !equivalentFileSystemSortValue(999, a, same) {
		t.Error("default: expected equivalent by ID")
	}
}

// ---------------------------------------------------------------------------
// nextFileSystemSortState
// ---------------------------------------------------------------------------

func TestNextFileSystemSortState(t *testing.T) {
	m := newTestModel(t)

	// First press on column 2 → asc.
	m.fsColumnSelected = 2
	m.fsSort = sortState{column: int(fsSortNone)}
	s := m.nextFileSystemSortState()
	if s.column != 2 || s.desc {
		t.Fatalf("expected col=2 asc, got col=%d desc=%v", s.column, s.desc)
	}

	// Second press same column → desc.
	m.fsSort = sortState{column: 2}
	s = m.nextFileSystemSortState()
	if s.column != 2 || !s.desc {
		t.Fatalf("expected col=2 desc, got col=%d desc=%v", s.column, s.desc)
	}

	// Third press → none.
	m.fsSort = sortState{column: 2, desc: true}
	s = m.nextFileSystemSortState()
	if s.column != int(fsSortNone) {
		t.Fatalf("expected fsSortNone, got %d", s.column)
	}

	// Different column resets.
	m.fsSort = sortState{column: 3, desc: true}
	m.fsColumnSelected = 5
	s = m.nextFileSystemSortState()
	if s.column != 5 || s.desc {
		t.Fatalf("expected col=5 asc, got col=%d desc=%v", s.column, s.desc)
	}
}

func TestNextSpaceSortState(t *testing.T) {
	m := newTestModel(t)

	m.spacesColumnSelected = 3
	m.spaceSort = sortState{column: int(spaceSortNone)}
	s := m.nextSpaceSortState()
	if s.column != 3 || s.desc {
		t.Fatalf("expected col=3 asc, got col=%d desc=%v", s.column, s.desc)
	}

	m.spaceSort = sortState{column: 3}
	s = m.nextSpaceSortState()
	if s.column != 3 || !s.desc {
		t.Fatalf("expected col=3 desc, got col=%d desc=%v", s.column, s.desc)
	}

	m.spaceSort = sortState{column: 3, desc: true}
	s = m.nextSpaceSortState()
	if s.column != int(spaceSortNone) {
		t.Fatalf("expected spaceSortNone, got %d", s.column)
	}
}

// ---------------------------------------------------------------------------
// nodeColumnIsEnum / fsColumnIsEnum
// ---------------------------------------------------------------------------

func TestNodeColumnIsEnum(t *testing.T) {
	m := newTestModel(t)
	enums := map[int]bool{
		int(fstFilterType):      true,
		int(fstFilterStatus):    true,
		int(fstFilterActivated): true,
	}
	for i := 0; i <= int(fstFilterType); i++ {
		got := m.nodeColumnIsEnum(i)
		if got != enums[i] {
			t.Errorf("column %d: expected %v, got %v", i, enums[i], got)
		}
	}
}

func TestFsColumnIsEnum(t *testing.T) {
	m := newTestModel(t)
	enums := map[int]bool{
		int(fsFilterBoot):         true,
		int(fsFilterConfigStatus): true,
		int(fsFilterDrain):        true,
		int(fsFilterStatus):       true,
	}
	for i := 0; i <= int(fsFilterHealth); i++ {
		got := m.fsColumnIsEnum(i)
		if got != enums[i] {
			t.Errorf("column %d: expected %v, got %v", i, enums[i], got)
		}
	}
}

// ---------------------------------------------------------------------------
// column counts
// ---------------------------------------------------------------------------

func TestFsColumnCount(t *testing.T) {
	if fsColumnCount() != 12 {
		t.Fatalf("expected 12, got %d", fsColumnCount())
	}
}

func TestGroupColumnCount(t *testing.T) {
	if groupColumnCount() != 7 {
		t.Fatalf("expected 7, got %d", groupColumnCount())
	}
}

func TestSpaceColumnCount(t *testing.T) {
	if spaceColumnCount() != 7 {
		t.Fatalf("expected 7, got %d", spaceColumnCount())
	}
}

// ---------------------------------------------------------------------------
// uniqueGroupValues
// ---------------------------------------------------------------------------

func TestUniqueGroupValues(t *testing.T) {
	m := newTestModel(t)
	m.groups = []eos.GroupRecord{
		{Name: "b", Status: "on"},
		{Name: "a", Status: "off"},
		{Name: "b", Status: "on"}, // duplicate
		{Name: "c", Status: "off"},
	}

	names := m.uniqueGroupValues(int(groupFilterName))
	if len(names) != 3 {
		t.Fatalf("expected 3 unique names, got %d", len(names))
	}
	// Should be sorted.
	if names[0] != "a" || names[1] != "b" || names[2] != "c" {
		t.Fatalf("expected [a,b,c], got %v", names)
	}

	statuses := m.uniqueGroupValues(int(groupFilterStatus))
	if len(statuses) != 2 {
		t.Fatalf("expected 2 unique statuses, got %d", len(statuses))
	}
}

func TestUniqueSpaceValues(t *testing.T) {
	m := newTestModel(t)
	m.spaces = []eos.SpaceRecord{
		{Name: "b", Status: "on"},
		{Name: "a", Status: "off"},
		{Name: "b", Status: "on"},
	}

	names := m.uniqueSpaceValues(int(spaceFilterName))
	if len(names) != 2 {
		t.Fatalf("expected 2 unique names, got %d", len(names))
	}
	if names[0] != "a" || names[1] != "b" {
		t.Fatalf("expected sorted names [a b], got %v", names)
	}
}

// ---------------------------------------------------------------------------
// Label functions
// ---------------------------------------------------------------------------

func TestFsSortStateLabel(t *testing.T) {
	m := newTestModel(t)

	// No sort → "none".
	m.fsSort = sortState{column: int(fsSortNone)}
	if got := m.fsSortStateLabel(); got != "none" {
		t.Fatalf("expected 'none', got %q", got)
	}

	// Ascending by host.
	m.fsSort = sortState{column: int(fsSortHost)}
	if got := m.fsSortStateLabel(); got != "host asc" {
		t.Fatalf("expected 'host asc', got %q", got)
	}

	// Descending by ID.
	m.fsSort = sortState{column: int(fsSortID), desc: true}
	if got := m.fsSortStateLabel(); got != "id desc" {
		t.Fatalf("expected 'id desc', got %q", got)
	}
}

func TestSpaceSortStateLabel(t *testing.T) {
	m := newTestModel(t)
	m.spaceSort = sortState{column: int(spaceSortNone)}
	if got := m.spaceSortStateLabel(); got != "none" {
		t.Fatalf("expected 'none', got %q", got)
	}

	m.spaceSort = sortState{column: int(spaceSortName)}
	if got := m.spaceSortStateLabel(); got != "name asc" {
		t.Fatalf("expected 'name asc', got %q", got)
	}

	m.spaceSort = sortState{column: int(spaceSortUsage), desc: true}
	if got := m.spaceSortStateLabel(); got != "usage % desc" {
		t.Fatalf("expected 'usage %% desc', got %q", got)
	}
}

func TestFstSortColumnLabelAll(t *testing.T) {
	m := newTestModel(t)
	cases := []struct {
		col   fstSortColumn
		label string
	}{
		{fstSortHost, "host"},
		{fstSortPort, "port"},
		{fstSortGeotag, "geotag"},
		{fstSortStatus, "status"},
		{fstSortActivated, "activated"},
		{fstSortHeartbeat, "heartbeatdelta"},
		{fstSortNoFS, "nofs"},
		{fstSortEOSVersion, "eos version"},
		{fstSortType, "type"},
		{fstSortNone, "none"},
	}
	for _, tc := range cases {
		m.fstSort.column = int(tc.col)
		if got := m.fstSortColumnLabel(); got != tc.label {
			t.Errorf("fstSort col %d: expected %q, got %q", tc.col, tc.label, got)
		}
	}

	// Default (unknown) → "host".
	m.fstSort.column = 999
	if got := m.fstSortColumnLabel(); got != "host" {
		t.Errorf("default: expected 'host', got %q", got)
	}
}

func TestFsFilterColumnLabelAll(t *testing.T) {
	m := newTestModel(t)
	cases := []struct {
		col   fsFilterColumn
		label string
	}{
		{fsFilterHost, "host"},
		{fsFilterPort, "port"},
		{fsFilterID, "id"},
		{fsFilterPath, "path"},
		{fsFilterGroup, "schedgroup"},
		{fsFilterGeotag, "geotag"},
		{fsFilterBoot, "boot"},
		{fsFilterConfigStatus, "configstatus"},
		{fsFilterDrain, "drain"},
		{fsFilterUsage, "usage %"},
		{fsFilterStatus, "active"},
		{fsFilterHealth, "health"},
	}
	for _, tc := range cases {
		m.fsFilter.column = int(tc.col)
		if got := m.fsFilterColumnLabel(); got != tc.label {
			t.Errorf("fsFilter col %d: expected %q, got %q", tc.col, tc.label, got)
		}
	}
	// Default (unknown) → "host".
	m.fsFilter.column = 999
	if got := m.fsFilterColumnLabel(); got != "host" {
		t.Errorf("default: expected 'host', got %q", got)
	}
}

func TestFsSortColumnLabelAll(t *testing.T) {
	m := newTestModel(t)
	cases := []struct {
		col   fsSortColumn
		label string
	}{
		{fsSortHost, "host"},
		{fsSortPort, "port"},
		{fsSortID, "id"},
		{fsSortPath, "path"},
		{fsSortGroup, "schedgroup"},
		{fsSortGeotag, "geotag"},
		{fsSortBoot, "boot"},
		{fsSortConfigStatus, "configstatus"},
		{fsSortDrain, "drain"},
		{fsSortUsed, "usage %"},
		{fsSortStatus, "active"},
		{fsSortHealth, "health"},
		{fsSortNone, "none"},
	}
	for _, tc := range cases {
		m.fsSort.column = int(tc.col)
		if got := m.fsSortColumnLabel(); got != tc.label {
			t.Errorf("fsSort col %d: expected %q, got %q", tc.col, tc.label, got)
		}
	}
	// Default.
	m.fsSort.column = 999
	if got := m.fsSortColumnLabel(); got != "host" {
		t.Errorf("default: expected 'host', got %q", got)
	}
}

func TestGroupFilterColumnLabelAll(t *testing.T) {
	m := newTestModel(t)
	cases := []struct {
		col   groupFilterColumn
		label string
	}{
		{groupFilterName, "name"},
		{groupFilterStatus, "status"},
		{groupFilterNoFS, "nofs"},
		{groupFilterCapacity, "capacity"},
		{groupFilterUsed, "used"},
		{groupFilterFree, "free"},
		{groupFilterFiles, "files"},
	}
	for _, tc := range cases {
		m.groupFilter.column = int(tc.col)
		if got := m.groupFilterColumnLabel(); got != tc.label {
			t.Errorf("groupFilter col %d: expected %q, got %q", tc.col, tc.label, got)
		}
	}
	m.groupFilter.column = 999
	if got := m.groupFilterColumnLabel(); got != "name" {
		t.Errorf("default: expected 'name', got %q", got)
	}
}

func TestSpaceFilterColumnLabelAll(t *testing.T) {
	m := newTestModel(t)
	cases := []struct {
		col   spaceFilterColumn
		label string
	}{
		{spaceFilterName, "name"},
		{spaceFilterType, "type"},
		{spaceFilterStatus, "status"},
		{spaceFilterGroups, "groups"},
		{spaceFilterFiles, "files"},
		{spaceFilterDirs, "dirs"},
		{spaceFilterUsage, "usage %"},
	}
	for _, tc := range cases {
		m.spaceFilter.column = int(tc.col)
		if got := m.spaceFilterColumnLabel(); got != tc.label {
			t.Errorf("spaceFilter col %d: expected %q, got %q", tc.col, tc.label, got)
		}
	}
	m.spaceFilter.column = 999
	if got := m.spaceFilterColumnLabel(); got != "name" {
		t.Errorf("default: expected 'name', got %q", got)
	}
}

func TestSpaceSortColumnLabelAll(t *testing.T) {
	m := newTestModel(t)
	cases := []struct {
		col   spaceSortColumn
		label string
	}{
		{spaceSortName, "name"},
		{spaceSortType, "type"},
		{spaceSortStatus, "status"},
		{spaceSortGroups, "groups"},
		{spaceSortFiles, "files"},
		{spaceSortDirs, "dirs"},
		{spaceSortUsage, "usage %"},
		{spaceSortNone, "none"},
	}
	for _, tc := range cases {
		m.spaceSort.column = int(tc.col)
		if got := m.spaceSortColumnLabel(); got != tc.label {
			t.Errorf("spaceSort col %d: expected %q, got %q", tc.col, tc.label, got)
		}
	}
	m.spaceSort.column = 999
	if got := m.spaceSortColumnLabel(); got != "name" {
		t.Errorf("default: expected 'name', got %q", got)
	}
}

func TestGroupSortColumnLabelAll(t *testing.T) {
	m := newTestModel(t)
	cases := []struct {
		col   groupSortColumn
		label string
	}{
		{groupSortName, "name"},
		{groupSortStatus, "status"},
		{groupSortNoFS, "nofs"},
		{groupSortCapacity, "capacity"},
		{groupSortUsed, "used"},
		{groupSortFree, "free"},
		{groupSortFiles, "files"},
		{groupSortNone, "none"},
	}
	for _, tc := range cases {
		m.groupSort.column = int(tc.col)
		if got := m.groupSortColumnLabel(); got != tc.label {
			t.Errorf("groupSort col %d: expected %q, got %q", tc.col, tc.label, got)
		}
	}
	m.groupSort.column = 999
	if got := m.groupSortColumnLabel(); got != "name" {
		t.Errorf("default: expected 'name', got %q", got)
	}
}

// ---------------------------------------------------------------------------
// activeFilterColumnLabel
// ---------------------------------------------------------------------------

func TestActiveFilterColumnLabelForDifferentViews(t *testing.T) {
	m := newTestModel(t)

	// FileSystems view.
	m.activeView = viewFileSystems
	m.fsFilter.column = int(fsFilterGeotag)
	if got := m.activeFilterColumnLabel(); got != "geotag" {
		t.Errorf("FS view: expected 'geotag', got %q", got)
	}

	// Groups view.
	m.activeView = viewGroups
	m.groupFilter.column = int(groupFilterStatus)
	if got := m.activeFilterColumnLabel(); got != "status" {
		t.Errorf("Groups view: expected 'status', got %q", got)
	}

	// Spaces view.
	m.activeView = viewSpaces
	m.spaceFilter.column = int(spaceFilterUsage)
	if got := m.activeFilterColumnLabel(); got != "usage %" {
		t.Errorf("Spaces view: expected 'usage %%', got %q", got)
	}

	// Default (FST) view.
	m.activeView = viewFST
	m.fstFilter.column = int(fstFilterEOSVersion)
	if got := m.activeFilterColumnLabel(); got != "eos version" {
		t.Errorf("FST view: expected 'eos version', got %q", got)
	}
}
