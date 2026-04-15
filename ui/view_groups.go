package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lobis/eos-tui/eos"
)

func (m model) renderGroupsView(height int) string {
	naturalListContent := 3 + len(m.visibleGroups())
	const groupDetailContent = 6
	listHeight, detailHeight := adaptiveSplitHeights(height, naturalListContent, groupDetailContent)

	width := m.panelWidth()
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

	title := m.renderSectionTitle("EOS Groups", contentWidth)
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
		m.renderSectionTitle("Selected Group", panelContentWidth(width)),
		g.Name,
		"",
		m.metricLine("Status", g.Status, "Filesystems", fmt.Sprintf("%d", g.NoFS)),
		m.metricLine("Capacity", humanBytes(g.CapacityBytes), "Used", humanBytes(g.UsedBytes)),
		m.metricLine("Free", humanBytes(g.FreeBytes), "Files", fmt.Sprintf("%d", g.NumFiles)),
	}

	return m.styles.panelDim.Width(width).Render(fitLines(lines, panelContentHeight(height)))
}

func (m model) selectedGroup() (eos.GroupRecord, bool) {
	groups := m.visibleGroups()
	if len(groups) == 0 || m.groupsSelected < 0 || m.groupsSelected >= len(groups) {
		return eos.GroupRecord{}, false
	}
	return groups[m.groupsSelected], true
}

func (m model) startGroupDrainConfirm() (tea.Model, tea.Cmd) {
	group, ok := m.selectedGroup()
	if !ok {
		return m, nil
	}

	m.groupDrain = groupDrainConfirm{
		active:  true,
		group:   group.Name,
		command: fmt.Sprintf("eos group set %s drain", group.Name),
		button:  buttonCancel,
	}
	return m, nil
}

func (m model) renderGroupDrainConfirmPopup() string {
	cancelBtn := "[ Cancel ]"
	confirmBtn := "[ Confirm ]"

	if m.groupDrain.button == buttonCancel {
		cancelBtn = m.styles.selected.Render(cancelBtn)
	} else {
		confirmBtn = m.styles.selected.Render(confirmBtn)
	}

	lines := []string{
		m.styles.popupTitle.Render("Confirm Group Drain"),
		fmt.Sprintf("Group: %s", m.styles.value.Render(m.groupDrain.group)),
		"",
		"The following command will be executed:",
		"",
		m.styles.value.Render(m.groupDrain.command),
		"",
		lipgloss.JoinHorizontal(lipgloss.Left, cancelBtn, "  ", confirmBtn),
		"",
		m.styles.status.Render("g cancel  •  G confirm  •  enter apply  •  esc close"),
	}

	return m.styles.panel.
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}
