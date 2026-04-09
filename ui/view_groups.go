package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderGroupsView(height int) string {
	const groupDetailLines = 6
	listHeight := max(4, height-groupDetailLines)
	detailHeight := groupDetailLines

	width := m.contentWidth()
	list := m.renderGroupsList(width, listHeight)
	details := m.renderGroupDetails(width, detailHeight)

	return lipgloss.JoinVertical(lipgloss.Left, list, details)
}

func (m model) renderGroupsList(width, height int) string {
	contentWidth := panelContentWidth(width)

	groups := m.visibleGroups()
	dataRows := make([][]string, len(groups))
	for i, g := range groups {
		dataRows[i] = []string{
			g.Name,
			g.Status,
			fmt.Sprintf("%d", g.NoFS),
			humanBytes(g.CapacityBytes),
			humanBytes(g.UsedBytes),
			humanBytes(g.FreeBytes),
			fmt.Sprintf("%d", g.NumFiles),
		}
	}

	columnDefs := contentAwareColumns([]tableColumn{
		{title: "name", min: 10, weight: 3},
		{title: "status", min: 6, weight: 1},
		{title: "nofs", min: 4, weight: 0, right: true},
		{title: "capacity", min: 8, weight: 0, right: true},
		{title: "used", min: 8, weight: 0, right: true},
		{title: "free", min: 8, weight: 0, right: true},
		{title: "files", min: 5, weight: 0, right: true},
	}, dataRows)

	columns := allocateTableColumns(contentWidth, columnDefs)

	title := m.styles.label.Render("EOS Groups")
	lines := []string{
		title,
		"",
		m.renderGroupHeaderRow(columns),
	}

	if m.groupsLoading {
		lines = append(lines, "Loading groups...")
	} else if m.groupsErr != nil {
		lines = append(lines, m.styles.error.Render(m.groupsErr.Error()))
	} else if len(groups) == 0 {
		lines = append(lines, "(no groups)")
	} else {
		start, end := visibleWindow(len(groups), m.groupsSelected, max(1, panelContentHeight(height)-len(lines)))
		lines[0] = title + renderScrollSummary(start, end, len(groups))
		for i := start; i < end; i++ {
			g := groups[i]
			row := []string{
				g.Name,
				g.Status,
				fmt.Sprintf("%d", g.NoFS),
				humanBytes(g.CapacityBytes),
				humanBytes(g.UsedBytes),
				humanBytes(g.FreeBytes),
				fmt.Sprintf("%d", g.NumFiles),
			}
			line := formatTableRow(columns, row)
			if i == m.groupsSelected {
				line = m.styles.selected.Width(contentWidth).Render(line)
			}
			lines = append(lines, line)
		}
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, panelContentHeight(height)))
}

func (m model) renderGroupHeaderRow(columns []tableColumn) string {
	labels := []string{"name", "status", "nofs", "capacity", "used", "free", "files"}
	return m.renderSelectableHeaderRow(columns, labels, m.groupsColumnSelected, m.groupSort, m.groupFilter)
}

func (m model) renderGroupDetails(width, height int) string {
	groups := m.visibleGroups()
	if len(groups) == 0 || m.groupsSelected < 0 || m.groupsSelected >= len(groups) {
		return m.styles.panelDim.Width(width).Render(fitLines([]string{"no group selected"}, panelContentHeight(height)))
	}

	g := groups[m.groupsSelected]
	lines := []string{
		m.styles.label.Render("Selected Group") + " " + g.Name,
		"",
		m.metricLine("Status", g.Status, "Filesystems", fmt.Sprintf("%d", g.NoFS)),
		m.metricLine("Capacity", humanBytes(g.CapacityBytes), "Used", humanBytes(g.UsedBytes)),
		m.metricLine("Free", humanBytes(g.FreeBytes), "Files", fmt.Sprintf("%d", g.NumFiles)),
	}

	return m.styles.panelDim.Width(width).Render(fitLines(lines, panelContentHeight(height)))
}
