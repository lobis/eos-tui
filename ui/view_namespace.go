package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/lobis/eos-tui/eos"
)

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

func (m model) selectedNamespaceEntry() (eos.Entry, bool) {
	if len(m.directory.Entries) == 0 || m.nsSelected < 0 || m.nsSelected >= len(m.directory.Entries) {
		return eos.Entry{}, false
	}

	return m.directory.Entries[m.nsSelected], true
}

func (m model) renderNamespaceHeaderRow(columns []tableColumn) string {
	return m.renderSimpleHeaderRow(columns, []string{"type", "name", "size", "uid", "gid", "modified"})
}
