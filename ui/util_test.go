package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/lobis/eos-tui/eos"
)

func TestCleanPathUI(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", "/"},
		{"root", "/", "/"},
		{"relative", "foo/bar", "/foo/bar"},
		{"absolute", "/foo/bar", "/foo/bar"},
		{"trailing slash", "/foo/bar/", "/foo/bar"},
		{"double slash", "/foo//bar", "/foo/bar"},
		{"dots", "/foo/bar/../baz", "/foo/baz"},
		{"already clean", "/a/b/c", "/a/b/c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanPath(tt.in)
			if got != tt.want {
				t.Fatalf("cleanPath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParentPath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"root stays root", "/", "/"},
		{"single level", "/foo", "/"},
		{"nested", "/foo/bar", "/foo"},
		{"deeply nested", "/a/b/c/d", "/a/b/c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parentPath(tt.in)
			if got != tt.want {
				t.Fatalf("parentPath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestResolveNamespacePath(t *testing.T) {
	tests := []struct {
		name    string
		current string
		input   string
		want    string
	}{
		{"empty input keeps current", "/eos/foo", "", "/eos/foo"},
		{"whitespace input keeps current", "/eos/foo", "   ", "/eos/foo"},
		{"absolute replaces", "/eos/foo", "/eos/bar/baz", "/eos/bar/baz"},
		{"absolute is cleaned", "/eos/foo", "/eos//bar/../baz", "/eos/baz"},
		{"relative joined onto current", "/eos/foo", "bar", "/eos/foo/bar"},
		{"dot dot walks up", "/eos/foo/bar", "..", "/eos/foo"},
		{"double dot dot walks up twice", "/eos/foo/bar", "../..", "/eos"},
		{"current empty treated as root", "", "foo", "/foo"},
		{"trailing slash trimmed", "/eos", "foo/bar/", "/eos/foo/bar"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveNamespacePath(tt.current, tt.input)
			if got != tt.want {
				t.Fatalf("resolveNamespacePath(%q, %q) = %q, want %q", tt.current, tt.input, got, tt.want)
			}
		})
	}
}

func TestFallback(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		defaultValue string
		want         string
	}{
		{"empty uses default", "", "default", "default"},
		{"non-empty uses value", "value", "default", "value"},
		{"both empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fallback(tt.value, tt.defaultValue)
			if got != tt.want {
				t.Fatalf("fallback(%q, %q) = %q, want %q", tt.value, tt.defaultValue, got, tt.want)
			}
		})
	}
}

func TestHumanBytesUI(t *testing.T) {
	tests := []struct {
		name string
		in   uint64
		want string
	}{
		{"zero", 0, "0 B"},
		{"one byte", 1, "1 B"},
		{"1023 bytes", 1023, "1023 B"},
		{"1 KiB", 1024, "1.0 KiB"},
		{"1.5 KiB", 1536, "1.5 KiB"},
		{"1 MiB", 1024 * 1024, "1.0 MiB"},
		{"1 GiB", 1024 * 1024 * 1024, "1.0 GiB"},
		{"1 TiB", 1024 * 1024 * 1024 * 1024, "1.0 TiB"},
		{"1 PiB", 1024 * 1024 * 1024 * 1024 * 1024, "1.0 PiB"},
		{"large PiB", 2 * 1024 * 1024 * 1024 * 1024 * 1024, "2.0 PiB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := humanBytes(tt.in)
			if got != tt.want {
				t.Fatalf("humanBytes(%d) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want string
	}{
		{"zero", 0, "-"},
		{"negative", -time.Second, "-"},
		{"one second", time.Second, "1s"},
		{"one minute", time.Minute, "1m0s"},
		{"complex", 3*time.Hour + 25*time.Minute + 10*time.Second, "3h25m10s"},
		{"sub-second rounds", 1500 * time.Millisecond, "2s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.in)
			if got != tt.want {
				t.Fatalf("formatDuration(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	t.Run("zero time", func(t *testing.T) {
		got := formatTime(time.Time{})
		if got != "-" {
			t.Fatalf("formatTime(zero) = %q, want %q", got, "-")
		}
	})

	t.Run("non-zero time", func(t *testing.T) {
		ts := time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC)
		got := formatTime(ts)
		if got == "-" {
			t.Fatal("formatTime(non-zero) returned dash")
		}
		// Must parse as RFC3339.
		if _, err := time.Parse(time.RFC3339, got); err != nil {
			t.Fatalf("formatTime output %q is not valid RFC3339: %v", got, err)
		}
	})
}

func TestFormatTimeShort(t *testing.T) {
	t.Run("zero time", func(t *testing.T) {
		got := formatTimeShort(time.Time{})
		if got != "-" {
			t.Fatalf("formatTimeShort(zero) = %q, want %q", got, "-")
		}
	})

	t.Run("non-zero time", func(t *testing.T) {
		ts := time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC)
		got := formatTimeShort(ts)
		if got == "-" {
			t.Fatal("formatTimeShort(non-zero) returned dash")
		}
		// The short format is "2006-01-02 15:04" — 16 chars.
		if len(got) != 16 {
			t.Fatalf("formatTimeShort output %q has unexpected length %d", got, len(got))
		}
	})
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name  string
		value string
		width int
		want  string
	}{
		{"fits", "hello", 10, "hello"},
		{"exact fit", "hello", 5, "hello"},
		{"exceeds", "hello world", 5, "hell…"},
		{"width zero", "hello", 0, "hello"},
		{"width one", "hello", 1, "…"},
		{"width negative", "hello", -1, "hello"},
		{"newlines replaced", "a\nb\nc", 10, "a b c"},
		{"empty string", "", 5, ""},
		{"truncate with newlines", "hello\nworld", 6, "hello…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.value, tt.width)
			if got != tt.want {
				t.Fatalf("truncate(%q, %d) = %q, want %q", tt.value, tt.width, got, tt.want)
			}
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name  string
		value string
		width int
		want  string
	}{
		{"shorter", "hi", 5, "hi   "},
		{"exact", "hello", 5, "hello"},
		{"longer truncated", "hello world", 5, "hell…"},
		{"empty", "", 5, "     "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := padRight(tt.value, tt.width)
			if got != tt.want {
				t.Fatalf("padRight(%q, %d) = %q, want %q", tt.value, tt.width, got, tt.want)
			}
		})
	}
}

func TestPadLeft(t *testing.T) {
	tests := []struct {
		name  string
		value string
		width int
		want  string
	}{
		{"shorter", "hi", 5, "   hi"},
		{"exact", "hello", 5, "hello"},
		{"longer truncated", "hello world", 5, "hell…"},
		{"empty", "", 5, "     "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := padLeft(tt.value, tt.width)
			if got != tt.want {
				t.Fatalf("padLeft(%q, %d) = %q, want %q", tt.value, tt.width, got, tt.want)
			}
		})
	}
}

func TestFormatTableRow(t *testing.T) {
	t.Run("normal row", func(t *testing.T) {
		cols := []tableColumn{
			{title: "Name", min: 10},
			{title: "Size", min: 8},
		}
		got := formatTableRow(cols, []string{"foo", "1 KiB"})
		if !strings.HasPrefix(got, "foo") {
			t.Fatalf("expected row to start with foo, got %q", got)
		}
	})

	t.Run("fewer values than columns", func(t *testing.T) {
		cols := []tableColumn{
			{title: "A", min: 5},
			{title: "B", min: 5},
			{title: "C", min: 5},
		}
		got := formatTableRow(cols, []string{"x"})
		parts := strings.SplitN(got, " ", 3)
		if len(parts) < 3 {
			t.Fatalf("expected 3 parts, got %d: %q", len(parts), got)
		}
	})

	t.Run("right aligned", func(t *testing.T) {
		cols := []tableColumn{
			{title: "Val", min: 6, right: true},
		}
		got := formatTableRow(cols, []string{"hi"})
		if !strings.HasPrefix(got, "    ") {
			t.Fatalf("expected leading spaces for right-aligned, got %q", got)
		}
	})
}

func TestAllocateTableColumns(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := allocateTableColumns(80, nil)
		if result != nil {
			t.Fatalf("expected nil, got %v", result)
		}
	})

	t.Run("sufficient width", func(t *testing.T) {
		cols := []tableColumn{
			{title: "A", min: 5, weight: 1},
			{title: "B", min: 5, weight: 1},
		}
		result := allocateTableColumns(80, cols)
		if len(result) != 2 {
			t.Fatalf("expected 2 columns, got %d", len(result))
		}
		total := result[0].min + result[1].min
		// With separators the available space is width-(n-1) = 79
		if total > 79 {
			t.Fatalf("total allocation %d exceeds available 79", total)
		}
	})

	t.Run("overflow", func(t *testing.T) {
		cols := []tableColumn{
			{title: "A", min: 50},
			{title: "B", min: 50},
		}
		result := allocateTableColumns(20, cols)
		if len(result) != 2 {
			t.Fatalf("expected 2 columns, got %d", len(result))
		}
		total := result[0].min + result[1].min
		if total > 19 {
			t.Fatalf("overflow: total %d > 19", total)
		}
	})

	t.Run("weighted distribution", func(t *testing.T) {
		cols := []tableColumn{
			{title: "A", min: 5, weight: 3},
			{title: "B", min: 5, weight: 1},
		}
		result := allocateTableColumns(100, cols)
		if result[0].min <= result[1].min {
			t.Fatalf("expected col A wider than B: A=%d B=%d", result[0].min, result[1].min)
		}
	})

	t.Run("maxw respected", func(t *testing.T) {
		cols := []tableColumn{
			{title: "A", min: 5, weight: 1, maxw: 10},
			{title: "B", min: 5, weight: 1},
		}
		result := allocateTableColumns(200, cols)
		if result[0].min > 10 {
			t.Fatalf("expected col A capped at 10, got %d", result[0].min)
		}
	})
}

func TestContentAwareColumns(t *testing.T) {
	t.Run("expands min to content width", func(t *testing.T) {
		cols := []tableColumn{
			{title: "N", min: 2},
		}
		rows := [][]string{
			{"hello"},
			{"world!"},
		}
		result := contentAwareColumns(cols, rows)
		if result[0].min < 6 {
			t.Fatalf("expected min >= 6, got %d", result[0].min)
		}
	})

	t.Run("respects maxw", func(t *testing.T) {
		cols := []tableColumn{
			{title: "N", min: 2, maxw: 4},
		}
		rows := [][]string{
			{"very long content here"},
		}
		result := contentAwareColumns(cols, rows)
		if result[0].min > 4 {
			t.Fatalf("expected min <= 4, got %d", result[0].min)
		}
	})

	t.Run("uses title width", func(t *testing.T) {
		cols := []tableColumn{
			{title: "LongTitle", min: 2},
		}
		result := contentAwareColumns(cols, nil)
		if result[0].min < len("LongTitle") {
			t.Fatalf("expected min >= title width, got %d", result[0].min)
		}
	})
}

func TestVisibleWindow(t *testing.T) {
	tests := []struct {
		name                 string
		total, selected, cap int
		wantStart, wantEnd   int
	}{
		{"all fit", 5, 2, 10, 0, 5},
		{"selected in middle", 20, 10, 10, 5, 15},
		{"selected at start", 20, 0, 10, 0, 10},
		{"selected at end", 20, 19, 10, 10, 20},
		{"negative selected", 20, -1, 10, 0, 10},
		{"empty total", 0, 0, 10, 0, 0},
		{"zero capacity", 10, 5, 0, 0, 10},
		{"selected beyond total", 10, 15, 5, 5, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := visibleWindow(tt.total, tt.selected, tt.cap)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Fatalf("visibleWindow(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.total, tt.selected, tt.cap, start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestRenderScrollSummary(t *testing.T) {
	tests := []struct {
		name              string
		start, end, total int
		want              string
	}{
		{"all visible", 0, 10, 10, ""},
		{"all visible zero", 0, 0, 0, ""},
		{"partial", 5, 15, 20, "  [6-15/20]"},
		{"beginning", 0, 10, 20, "  [1-10/20]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderScrollSummary(tt.start, tt.end, tt.total)
			if got != tt.want {
				t.Fatalf("renderScrollSummary(%d, %d, %d) = %q, want %q",
					tt.start, tt.end, tt.total, got, tt.want)
			}
		})
	}
}

func TestPanelContentWidth(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"normal", 80, 76},
		{"small", 5, 1},
		{"zero", 0, 1},
		{"negative", -5, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := panelContentWidth(tt.in)
			if got != tt.want {
				t.Fatalf("panelContentWidth(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestPanelContentHeight(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"normal", 24, 22},
		{"small", 3, 1},
		{"zero", 0, 1},
		{"negative", -5, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := panelContentHeight(tt.in)
			if got != tt.want {
				t.Fatalf("panelContentHeight(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestFitLines(t *testing.T) {
	t.Run("fewer lines than height", func(t *testing.T) {
		got := fitLines([]string{"a", "b"}, 4)
		lines := strings.Split(got, "\n")
		if len(lines) != 4 {
			t.Fatalf("expected 4 lines, got %d", len(lines))
		}
	})

	t.Run("more lines than height", func(t *testing.T) {
		got := fitLines([]string{"a", "b", "c", "d", "e"}, 3)
		lines := strings.Split(got, "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d", len(lines))
		}
	})

	t.Run("zero height", func(t *testing.T) {
		got := fitLines([]string{"a"}, 0)
		if got != "" {
			t.Fatalf("expected empty string, got %q", got)
		}
	})

	t.Run("exact height", func(t *testing.T) {
		got := fitLines([]string{"a", "b", "c"}, 3)
		lines := strings.Split(got, "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d", len(lines))
		}
	})
}

func TestSplitViewHeights(t *testing.T) {
	totals := []int{1, 2, 3, 10, 20, 40}
	for _, total := range totals {
		list, detail := splitViewHeights(total)
		if list < 1 {
			t.Fatalf("splitViewHeights(%d): list=%d < 1", total, list)
		}
		if detail < 1 {
			t.Fatalf("splitViewHeights(%d): detail=%d < 1", total, detail)
		}
	}

	// For a reasonable size, list should be larger than detail.
	list, detail := splitViewHeights(40)
	if list <= detail {
		t.Fatalf("splitViewHeights(40): expected list > detail, got list=%d detail=%d", list, detail)
	}
}

func TestAdaptiveSplitHeights(t *testing.T) {
	t.Run("everything fits", func(t *testing.T) {
		list, detail := adaptiveSplitHeights(30, 10, 5)
		if list+detail != 32 { // target = height + 2
			t.Fatalf("total should be 32, got %d", list+detail)
		}
		if detail != 7 { // naturalDetail = content + 2
			t.Fatalf("detail should be 7 (natural), got %d", detail)
		}
	})

	t.Run("tight space", func(t *testing.T) {
		list, detail := adaptiveSplitHeights(12, 20, 20)
		if list < 4 {
			t.Fatalf("list should be at least 4, got %d", list)
		}
		if list+detail != 14 { // target = 12 + 2
			t.Fatalf("total should be 14, got %d", list+detail)
		}
	})

	t.Run("very tight space", func(t *testing.T) {
		list, _ := adaptiveSplitHeights(6, 20, 20)
		if list < 4 {
			t.Fatalf("list should be at least 4, got %d", list)
		}
	})
}

func TestUsagePercent(t *testing.T) {
	tests := []struct {
		name      string
		used, cap uint64
		want      float64
	}{
		{"normal", 50, 100, 50.0},
		{"full", 100, 100, 100.0},
		{"zero capacity", 50, 0, 0.0},
		{"zero used", 0, 100, 0.0},
		{"half", 1, 2, 50.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := usagePercent(tt.used, tt.cap)
			if got != tt.want {
				t.Fatalf("usagePercent(%d, %d) = %f, want %f", tt.used, tt.cap, got, tt.want)
			}
		})
	}
}

func TestMinMax(t *testing.T) {
	if max(3, 5) != 5 {
		t.Fatal("max(3,5) != 5")
	}
	if max(5, 3) != 5 {
		t.Fatal("max(5,3) != 5")
	}
	if max(3, 3) != 3 {
		t.Fatal("max(3,3) != 3")
	}
	if min(3, 5) != 3 {
		t.Fatal("min(3,5) != 3")
	}
	if min(5, 3) != 3 {
		t.Fatal("min(5,3) != 3")
	}
	if min(3, 3) != 3 {
		t.Fatal("min(3,3) != 3")
	}
	if max(-1, -2) != -1 {
		t.Fatal("max(-1,-2) != -1")
	}
	if min(-1, -2) != -2 {
		t.Fatal("min(-1,-2) != -2")
	}
}

func TestClampIndex(t *testing.T) {
	tests := []struct {
		name          string
		index, length int
		want          int
	}{
		{"in range", 3, 10, 3},
		{"at zero", 0, 10, 0},
		{"at last", 9, 10, 9},
		{"below zero", -1, 10, 0},
		{"above length", 15, 10, 9},
		{"zero length", 5, 0, 0},
		{"zero length neg", -1, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampIndex(tt.index, tt.length)
			if got != tt.want {
				t.Fatalf("clampIndex(%d, %d) = %d, want %d", tt.index, tt.length, got, tt.want)
			}
		})
	}
}

func TestMax64(t *testing.T) {
	if max64(3, 5) != 5 {
		t.Fatal("max64(3,5) != 5")
	}
	if max64(5, 3) != 5 {
		t.Fatal("max64(5,3) != 5")
	}
	if max64(3, 3) != 3 {
		t.Fatal("max64(3,3) != 3")
	}
	if max64(-10, -20) != -10 {
		t.Fatal("max64(-10,-20) != -10")
	}
}

func TestEntryTypeLabel(t *testing.T) {
	t.Run("container returns DIR", func(t *testing.T) {
		entry := eos.Entry{Kind: eos.EntryKindContainer}
		if got := entryTypeLabel(entry); got != "DIR" {
			t.Fatalf("entryTypeLabel(container) = %q, want DIR", got)
		}
	})

	t.Run("file returns FILE", func(t *testing.T) {
		entry := eos.Entry{Kind: eos.EntryKindFile}
		if got := entryTypeLabel(entry); got != "FILE" {
			t.Fatalf("entryTypeLabel(file) = %q, want FILE", got)
		}
	})
}

func TestEntrySize(t *testing.T) {
	t.Run("container returns dash", func(t *testing.T) {
		entry := eos.Entry{Kind: eos.EntryKindContainer}
		if got := entrySize(entry); got != "-" {
			t.Fatalf("entrySize(container) = %q, want '-'", got)
		}
	})

	t.Run("file returns formatted size", func(t *testing.T) {
		entry := eos.Entry{Kind: eos.EntryKindFile, Size: 2048}
		got := entrySize(entry)
		if got != "2.0 KiB" {
			t.Fatalf("entrySize(file 2048) = %q, want '2.0 KiB'", got)
		}
	})

	t.Run("file zero bytes", func(t *testing.T) {
		entry := eos.Entry{Kind: eos.EntryKindFile, Size: 0}
		got := entrySize(entry)
		if got != "0 B" {
			t.Fatalf("entrySize(file 0) = %q, want '0 B'", got)
		}
	})
}
