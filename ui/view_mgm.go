package ui

import (
	"fmt"
	"strings"
)

// renderMGMView shows the unified topology as two stacked tables:
// management nodes first, then the matching QDB hosts.
func (m model) renderMGMView(height int) string {
	width := m.panelWidth()
	contentWidth := panelContentWidth(width)
	mgms := m.mgms

	title := m.styles.label.Render("management & quarkdb topology")
	lines := []string{title, ""}

	if m.mgmsLoading && len(mgms) == 0 {
		lines = append(lines, "loading management and quarkdb info...")
	} else if m.mgmsErr != nil {
		lines = append(lines, m.styles.error.Render(m.mgmsErr.Error()))
	} else if len(mgms) == 0 {
		lines = append(lines, "(no management nodes found)")
	} else {
		availableRows := max(1, panelContentHeight(height)-6)
		topRows := max(1, availableRows/2)
		bottomRows := max(1, availableRows-topRows)

		lines = append(lines, m.renderSectionTitle("Management Nodes (MGM)", contentWidth))
		lines = append(lines, m.renderMGMTopologyRows(contentWidth, topRows)...)
		lines = append(lines, "")
		lines = append(lines, m.renderSectionTitle("QuarkDB Hosts (QDB)", contentWidth))
		lines = append(lines, m.renderQDBTopologyRows(contentWidth, bottomRows)...)
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, height))
}

func (m model) renderMGMTopologyRows(contentWidth, maxRows int) []string {
	columns := allocateTableColumns(contentWidth, []tableColumn{
		{title: "host", min: 15, weight: 1},
		{title: "port", min: 4, weight: 0, right: true},
		{title: "role", min: 10, weight: 0},
		{title: "status", min: 10, weight: 0},
		{title: "version", min: 10, weight: 0},
	})

	lines := []string{
		m.renderSimpleHeaderRow(columns, []string{"host", "port", "role", "status", "version"}),
	}

	for _, idx := range visibleTableIndices(len(m.mgms), m.mgmSelected, maxRows) {
		node := m.mgms[idx]
		version := node.EOSVersion
		if version == "" {
			version = m.eosVersion
		}
		row := formatTableRow(columns, []string{
			node.Host,
			fmt.Sprintf("%d", node.Port),
			strings.ToLower(node.Role),
			strings.ToLower(node.Status),
			version,
		})
		if idx == m.mgmSelected {
			row = m.styles.selected.Width(contentWidth).Render(row)
		}
		lines = append(lines, row)
	}

	return lines
}

func (m model) renderQDBTopologyRows(contentWidth, maxRows int) []string {
	columns := allocateTableColumns(contentWidth, []tableColumn{
		{title: "host", min: 15, weight: 1},
		{title: "port", min: 4, weight: 0, right: true},
		{title: "role", min: 10, weight: 0},
		{title: "status", min: 10, weight: 0},
		{title: "version", min: 10, weight: 0},
	})

	lines := []string{
		m.renderSimpleHeaderRow(columns, []string{"host", "port", "role", "status", "version"}),
	}

	for _, idx := range visibleTableIndices(len(m.mgms), m.mgmSelected, maxRows) {
		node := m.mgms[idx]
		version := node.EOSVersion
		if version == "" {
			version = m.eosVersion
		}
		row := formatTableRow(columns, []string{
			node.QDBHost,
			fmt.Sprintf("%d", node.QDBPort),
			strings.ToLower(node.Role),
			strings.ToLower(node.Status),
			version,
		})
		if idx == m.mgmSelected {
			row = m.styles.selected.Width(contentWidth).Render(row)
		}
		lines = append(lines, row)
	}

	return lines
}

func visibleTableIndices(total, selected, maxRows int) []int {
	if total <= 0 || maxRows <= 0 {
		return nil
	}
	if total <= maxRows {
		out := make([]int, total)
		for i := 0; i < total; i++ {
			out[i] = i
		}
		return out
	}

	start := max(0, selected-maxRows/2)
	if start+maxRows > total {
		start = total - maxRows
	}
	end := min(total, start+maxRows)

	out := make([]int, 0, end-start)
	for i := start; i < end; i++ {
		out = append(out, i)
	}
	return out
}
