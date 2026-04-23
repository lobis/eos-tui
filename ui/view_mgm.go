package ui

import (
	"fmt"
	"strings"
)

// renderMGMView shows the combined MGM and QDB topology in a single table.
func (m model) renderMGMView(height int) string {
	width := m.panelWidth()
	contentWidth := panelContentWidth(width)

	mgms := m.mgms

	columns := allocateTableColumns(contentWidth, []tableColumn{
		{title: "mgm host", min: 15, weight: 1},
		{title: "port", min: 4, weight: 0, right: true},
		{title: "qdb host", min: 15, weight: 1},
		{title: "port", min: 4, weight: 0, right: true},
		{title: "role", min: 10, weight: 0},
		{title: "status", min: 10, weight: 0},
		{title: "version", min: 10, weight: 0},
	})

	title := m.styles.label.Render("management & quarkdb topology")
	lines := []string{
		title,
		"",
		m.renderSimpleHeaderRow(columns, []string{"mgm host", "port", "qdb host", "port", "role", "status", "version"}),
	}

	if m.mgmsLoading && len(mgms) == 0 {
		lines = append(lines, "loading management and quarkdb info...")
	} else if m.mgmsErr != nil {
		lines = append(lines, m.styles.error.Render(m.mgmsErr.Error()))
	} else if len(mgms) == 0 {
		lines = append(lines, "(no management nodes found)")
	} else {
		for i, node := range mgms {
			version := node.EOSVersion
			if version == "" {
				version = m.eosVersion
			}
			row := formatTableRow(columns, []string{
				node.Host,
				fmt.Sprintf("%d", node.Port),
				node.QDBHost,
				fmt.Sprintf("%d", node.QDBPort),
				strings.ToLower(node.Role),
				strings.ToLower(node.Status),
				version,
			})
			if i == m.mgmSelected {
				row = m.styles.selected.Width(contentWidth).Render(row)
			}
			lines = append(lines, row)
		}
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, height))
}
