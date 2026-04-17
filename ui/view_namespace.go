package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/lobis/eos-tui/eos"
)

func (m model) renderNamespaceView(height int) string {
	width := m.panelWidth()

	fixedListLines := 3 // Title, blank, header
	naturalListContent := fixedListLines + len(m.directory.Entries)
	if !m.nsLoaded && !m.nsLoading {
		naturalListContent = 4 // Title, blank, header, "(empty)" or hint
	}

	naturalDetailContent := m.namespaceDetailContentTarget()

	listHeight, detailHeight := adaptiveSplitHeights(height, naturalListContent, naturalDetailContent)

	list := m.renderNamespaceList(width, listHeight)
	details := m.renderNamespaceDetails(width, detailHeight)
	return lipgloss.JoinVertical(lipgloss.Left, list, details)
}

func (m model) namespaceDetailContentTarget() int {
	return max(m.nsDetailContentMax, m.namespaceDetailContentCurrent())
}

func (m model) rememberNamespaceDetailContent() model {
	if current := m.namespaceDetailContentCurrent(); current > m.nsDetailContentMax {
		m.nsDetailContentMax = current
	}
	return m
}

func (m model) namespaceDetailContentCurrent() int {
	return max(m.namespaceMetadataContentLines(), m.namespaceAttrsContentLines())
}

func (m model) namespaceMetadataContentLines() int {
	target := m.directory.Self
	if selected, ok := m.selectedNamespaceEntry(); ok {
		target = selected
	}

	lines := 7 // title, path, blank, and four metric rows
	if target.Kind == eos.EntryKindContainer {
		lines += 2
	} else {
		lines += 2
		if target.LinkName != "" {
			lines++
		}
	}
	return lines
}

func (m model) namespaceAttrsContentLines() int {
	target := m.directory.Self
	if selected, ok := m.selectedNamespaceEntry(); ok {
		target = selected
	}

	lines := 3 // title, path, blank
	switch {
	case m.nsAttrsLoading && m.nsAttrsTargetPath == target.Path:
		lines++
	case m.nsAttrsErr != nil && m.nsAttrsTargetPath == target.Path:
		lines++
	case m.nsAttrsLoaded && m.nsAttrsTargetPath == target.Path && len(m.nsAttrs) == 0:
		lines++
	case m.nsAttrsLoaded && m.nsAttrsTargetPath == target.Path:
		lines += len(m.nsAttrs)
	default:
		lines++
	}
	return lines
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
	if width < 72 {
		return m.renderNamespaceMetadataPanel(width, height)
	}

	totalWidth := width + 2
	gap := 1
	availableWidth := max(1, totalWidth-gap)
	const minLeftWidth = 38
	const minRightWidth = 28

	leftWidth := max(minLeftWidth, availableWidth/2-2)
	rightWidth := availableWidth - leftWidth

	leftNaturalWidth := m.namespaceMetadataNaturalWidth()
	rightNaturalWidth := m.namespaceAttrsNaturalWidth()

	if leftNaturalWidth > leftWidth {
		grow := min(leftNaturalWidth-leftWidth, max(0, rightWidth-minRightWidth))
		leftWidth += grow
		rightWidth -= grow
	}
	if rightNaturalWidth > rightWidth {
		grow := min(rightNaturalWidth-rightWidth, max(0, leftWidth-minLeftWidth))
		rightWidth += grow
		leftWidth -= grow
	}
	if leftWidth < minLeftWidth {
		shift := min(minLeftWidth-leftWidth, max(0, rightWidth-minRightWidth))
		leftWidth += shift
		rightWidth -= shift
	}
	if rightWidth < minRightWidth {
		shift := min(minRightWidth-rightWidth, max(0, leftWidth-minLeftWidth))
		rightWidth += shift
		leftWidth -= shift
	}

	left := m.renderNamespaceMetadataPanel(leftWidth, height)
	right := m.renderNamespaceAttrsPanel(rightWidth, height)
	for i := 0; i < 4; i++ {
		combinedWidth := lipgloss.Width(left) + gap + lipgloss.Width(right)
		if combinedWidth <= totalWidth {
			break
		}

		overflow := combinedWidth - totalWidth
		if lipgloss.Width(left) >= lipgloss.Width(right) && leftWidth > minLeftWidth {
			shrink := min(overflow, leftWidth-minLeftWidth)
			leftWidth -= shrink
		} else if rightWidth > minRightWidth {
			shrink := min(overflow, rightWidth-minRightWidth)
			rightWidth -= shrink
		} else if leftWidth > minLeftWidth {
			shrink := min(overflow, leftWidth-minLeftWidth)
			leftWidth -= shrink
		} else {
			break
		}

		left = m.renderNamespaceMetadataPanel(leftWidth, height)
		right = m.renderNamespaceAttrsPanel(rightWidth, height)
	}

	combinedWidth := lipgloss.Width(left) + gap + lipgloss.Width(right)
	if deficit := totalWidth - combinedWidth; deficit > 0 {
		rightWidth += deficit
		right = m.renderNamespaceAttrsPanel(rightWidth, height)
	}

	return normalizeBlockWidth(lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right), totalWidth)
}

func (m model) namespaceMetadataNaturalWidth() int {
	target := m.directory.Self
	if selected, ok := m.selectedNamespaceEntry(); ok {
		target = selected
	}

	lines := []string{
		"Selected Namespace Entry",
		target.Path,
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

	maxWidth := 0
	for _, line := range lines {
		maxWidth = max(maxWidth, lipgloss.Width(line))
	}
	return maxWidth + 4
}

func (m model) namespaceAttrsNaturalWidth() int {
	target := m.directory.Self
	if selected, ok := m.selectedNamespaceEntry(); ok {
		target = selected
	}

	lines := []string{"Attributes", target.Path}
	switch {
	case m.nsAttrsLoading && m.nsAttrsTargetPath == target.Path:
		lines = append(lines, "Loading attributes...")
	case m.nsAttrsErr != nil && m.nsAttrsTargetPath == target.Path:
		lines = append(lines, m.nsAttrsErr.Error())
	case m.nsAttrsLoaded && m.nsAttrsTargetPath == target.Path && len(m.nsAttrs) == 0:
		lines = append(lines, "(no attributes)")
	default:
		for _, attr := range m.nsAttrs {
			lines = append(lines, fmt.Sprintf("%s = %s", attr.Key, attr.Value))
		}
	}

	maxWidth := 0
	for _, line := range lines {
		maxWidth = max(maxWidth, lipgloss.Width(line))
	}
	return maxWidth + 4
}

func (m model) renderNamespaceMetadataPanel(width, height int) string {
	contentWidth := panelContentWidth(width)
	target := m.directory.Self
	if selected, ok := m.selectedNamespaceEntry(); ok {
		target = selected
	}

	lines := []string{
		m.renderSectionTitle("Selected Namespace Entry", contentWidth),
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

	return m.styles.panelDim.Width(width).Render(normalizePanelLines(lines, contentWidth, panelContentHeight(height)))
}

func (m model) renderNamespaceAttrsPanel(width, height int) string {
	contentWidth := panelContentWidth(width)
	target := m.directory.Self
	if selected, ok := m.selectedNamespaceEntry(); ok {
		target = selected
	}

	lines := []string{
		m.renderSectionTitle("Attributes", contentWidth),
		truncate(target.Path, max(10, contentWidth)),
		"",
	}

	switch {
	case m.nsAttrsLoading && m.nsAttrsTargetPath == target.Path:
		lines = append(lines, "Loading attributes...")
	case m.nsAttrsErr != nil && m.nsAttrsTargetPath == target.Path:
		lines = append(lines, m.styles.error.Render(m.nsAttrsErr.Error()))
	case m.nsAttrsLoaded && m.nsAttrsTargetPath == target.Path && len(m.nsAttrs) == 0:
		lines = append(lines, "(no attributes)")
	default:
		for i := 0; i < len(m.nsAttrs); i++ {
			attr := m.nsAttrs[i]
			lines = append(lines, truncate(fmt.Sprintf("%s = %s", attr.Key, attr.Value), contentWidth))
		}
	}

	return m.styles.panelDim.Width(width).Render(normalizePanelLines(lines, contentWidth, panelContentHeight(height)))
}

func normalizePanelLines(lines []string, contentWidth, contentHeight int) string {
	fitted := fitLines(lines, contentHeight)
	rows := strings.Split(fitted, "\n")
	for i, row := range rows {
		rows[i] = padVisibleWidth(row, contentWidth)
	}
	return strings.Join(rows, "\n")
}

func normalizeBlockWidth(block string, width int) string {
	rows := strings.Split(block, "\n")
	for i, row := range rows {
		rows[i] = padVisibleWidth(row, width)
	}
	return strings.Join(rows, "\n")
}

func (m model) renderNamespaceAttrEditPopup() string {
	if len(m.nsAttrEdit.attrs) == 0 {
		return m.styles.panel.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Render("No attributes available")
	}

	current := m.nsAttrEdit.attrs[m.nsAttrEdit.selected]
	recursiveValue := "No"
	if m.nsAttrEdit.recursive {
		recursiveValue = "Yes"
	}
	lines := []string{
		m.styles.popupTitle.Render("Edit Attribute"),
		truncate(m.nsAttrEdit.targetPath, 72),
		"",
	}

	if m.nsAttrEdit.stage == attrEditStageSelect {
		lines = append(lines,
			fmt.Sprintf("Recursive: %s", m.styles.value.Render(recursiveValue)),
			"",
			m.renderSectionTitle("Select Key", 72),
		)
		for i, attr := range m.nsAttrEdit.attrs {
			line := truncate(fmt.Sprintf("%s = %s", attr.Key, attr.Value), 72)
			if i == m.nsAttrEdit.selected {
				lines = append(lines, m.styles.selected.Render("▶ "+line))
			} else {
				lines = append(lines, "  "+line)
			}
		}
		lines = append(lines, "", m.styles.status.Render("↑↓ select  •  g/G home/end  •  enter edit  •  r toggle recursive  •  esc cancel"))
	} else {
		lines = append(lines,
			fmt.Sprintf("Key:     %s", m.styles.value.Render(current.Key)),
			fmt.Sprintf("Current: %s", m.styles.value.Render(current.Value)),
			fmt.Sprintf("Recursive: %s", m.styles.value.Render(recursiveValue)),
			"",
			m.nsAttrEdit.input.View(),
			"",
			m.styles.status.Render("enter apply  •  r toggle recursive  •  esc cancel"),
		)
	}

	return m.styles.panel.
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
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
