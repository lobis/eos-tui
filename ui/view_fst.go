package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/lobis/eos-tui/eos"
)

func (m model) renderFSTView(height int) string {
	filterLines := 0
	if len(m.fstFilter.filters) > 0 {
		filterLines = 1
	}
	fixedHeaderLines := 6 + filterLines // title+controls, 3 metric lines, blank, col headers [, filters]
	naturalListContent := fixedHeaderLines + len(m.visibleFSTs())
	const fstDetailLines = 18 // fixed lines rendered by renderNodeDetails
	listHeight, detailHeight := adaptiveSplitHeights(height, naturalListContent, fstDetailLines)
	width := m.contentWidth()

	list := m.renderNodesList(width, listHeight)
	details := m.renderNodeDetails(width, detailHeight)
	return lipgloss.JoinVertical(lipgloss.Left, list, details)
}

func (m model) renderNodesList(width, height int) string {
	contentWidth := panelContentWidth(width)
	fsts := m.visibleFSTs()

	// Build data rows first so column widths can be fitted to content.
	dataRows := make([][]string, len(fsts))
	for i, node := range fsts {
		dataRows[i] = []string{
			node.Host,
			fmt.Sprintf("%d", node.Port),
			node.Geotag,
			node.Status,
			node.Activated,
			fmt.Sprintf("%d", node.HeartbeatDelta),
			fmt.Sprintf("%d", node.FileSystemCount),
			node.EOSVersion,
		}
	}
	columnDefs := contentAwareColumns([]tableColumn{
		{title: "host", min: 8, weight: 5},
		{title: "port", min: 5, weight: 0, right: true},
		{title: "geotag", min: 6, weight: 3},
		{title: "status", min: 6, weight: 0},
		{title: "activated", min: 9, weight: 0},
		{title: "heartbeatdelta", min: 14, weight: 0, right: true},
		{title: "nofs", min: 4, weight: 0, right: true},
		{title: "eos version", min: 11, weight: 0},
	}, dataRows)
	columns := allocateTableColumns(contentWidth, columnDefs)

	title := m.styles.label.Render("Cluster Summary")
	lines := []string{
		title + m.renderNodeControls(),
		m.metricLine("Health", fallback(m.nodeStats.State, "-"), "Threads", fmt.Sprintf("%d", m.nodeStats.ThreadCount)),
		m.metricLine("Files", fmt.Sprintf("%d", m.nodeStats.FileCount), "Dirs", fmt.Sprintf("%d", m.nodeStats.DirCount)),
		m.metricLine("Uptime", formatDuration(m.nodeStats.Uptime), "FDs", fmt.Sprintf("%d", m.nodeStats.FileDescs)),
		"",
		m.renderFstHeaderRow(columns),
	}

	if m.fstStatsLoading {
		lines[1] = m.styles.value.Render("Loading cluster summary...")
		lines[2] = ""
		lines[3] = ""
	}
	if summary := m.renderFilterSummary(m.fstFilter.filters, func(col int) string {
		old := m.fstFilter.column
		m.fstFilter.column = col
		label := m.fstFilterColumnLabel()
		m.fstFilter.column = old
		return label
	}); summary != "" {
		lines = append(lines, summary)
	}

	if m.fstsLoading {
		lines = append(lines, "Loading node list...")
	} else if m.fstsErr != nil {
		lines = append(lines, m.styles.error.Render(m.fstsErr.Error()))
	} else if len(fsts) == 0 {
		lines = append(lines, "(no fsts)")
	} else {
		start, end := visibleWindow(len(fsts), m.fstSelected, max(1, panelContentHeight(height)-len(lines)))
		lines[0] = title + m.renderNodeControls() + renderScrollSummary(start, end, len(fsts))
		for i := start; i < end; i++ {
			line := formatTableRow(columns, dataRows[i])
			if i == m.fstSelected {
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
		truncate(node.Host+":"+fmt.Sprintf("%d", node.Port), max(10, width-4)),
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

func (m model) selectedNode() (eos.FstRecord, bool) {
	fsts := m.visibleFSTs()
	if len(fsts) == 0 || m.fstSelected < 0 || m.fstSelected >= len(fsts) {
		return eos.FstRecord{}, false
	}

	return fsts[m.fstSelected], true
}

func (m model) renderFstHeaderRow(columns []tableColumn) string {
	labels := []string{"host", "port", "geotag", "status", "activated", "heartbeatdelta", "nofs", "eos version"}
	return m.renderSelectableHeaderRow(columns, labels, m.fstColumnSelected, m.fstSort, m.fstFilter)
}

func (m model) renderNodeControls() string {
	return fmt.Sprintf("  [col:%s filters:%d current:%s]",
		m.fstSelectedColumnLabel(),
		len(m.fstFilter.filters),
		filterValueLabel(m.fstFilter.filters[m.fstColumnSelected], m.popup.active && m.popup.view == viewFST, m.popup.input.Value()),
	)
}
