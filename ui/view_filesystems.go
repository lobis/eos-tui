package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lobis/eos-tui/eos"
)

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

func (m model) selectedFileSystem() (eos.FileSystemRecord, bool) {
	fileSystems := m.visibleFileSystems()
	if len(fileSystems) == 0 || m.fsSelected < 0 || m.fsSelected >= len(fileSystems) {
		return eos.FileSystemRecord{}, false
	}

	return fileSystems[m.fsSelected], true
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

func (m model) renderFileSystemHeaderRow(columns []tableColumn) string {
	labels := []string{"host", "port", "id", "path", "schedgroup", "geotag", "boot", "configstatus", "drain", "usage %", "active", "health"}
	return m.renderSelectableHeaderRow(columns, labels, m.fsColumnSelected, m.fsSort, m.fsFilter)
}

func (m model) renderFileSystemControls() string {
	return fmt.Sprintf("  [col:%s filters:%d current:%s]",
		m.fsSelectedColumnLabel(),
		len(m.fsFilter.filters),
		filterValueLabel(m.fsFilter.filters[m.fsColumnSelected], m.popup.active && m.popup.view == viewFileSystems, m.popup.input.Value()),
	)
}
