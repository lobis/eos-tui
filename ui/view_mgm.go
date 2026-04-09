package ui

import (
	"fmt"
	"strings"
)

// renderMGMView shows the MGM nodes with their EOS server version.
// The MGM hosts are the same nodes that participate in the QDB cluster.
// The EOS server version is fetched via `eos version` (applies to all MGMs).
func (m model) renderMGMView(height int) string {
	width := m.contentWidth()
	contentWidth := panelContentWidth(width)

	mgms := m.mgms

	columns := allocateTableColumns(contentWidth, []tableColumn{
		{title: "host", min: 15, weight: 1},
		{title: "port", min: 5, weight: 0, right: true},
		{title: "role", min: 10, weight: 0},
		{title: "status", min: 10, weight: 0},
		{title: "eos version", min: 14, weight: 0},
	})

	title := m.styles.label.Render("management nodes (mgm)")
	lines := []string{
		title,
		"",
		m.renderSimpleHeaderRow(columns, []string{"host", "port", "role", "status", "eos version"}),
	}

	if m.mgmsLoading && len(mgms) == 0 {
		lines = append(lines, "loading mgm info...")
	} else if len(mgms) == 0 {
		lines = append(lines, "(no mgm nodes found)")
	} else {
		for i, node := range mgms {
			row := formatTableRow(columns, []string{
				node.Host,
				fmt.Sprintf("%d", node.Port),
				strings.ToLower(node.Role),
				strings.ToLower(node.Status),
				m.eosVersion,
			})
			if i == m.mgmSelected {
				row = m.styles.selected.Width(contentWidth).Render(row)
			}
			lines = append(lines, row)
		}
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, height))
}

// renderQDBView shows the QDB cluster topology from `redis-cli raft-info`.
func (m model) renderQDBView(height int) string {
	width := m.contentWidth()
	contentWidth := panelContentWidth(width)

	mgms := m.mgms

	columns := allocateTableColumns(contentWidth, []tableColumn{
		{title: "host", min: 15, weight: 1},
		{title: "port", min: 5, weight: 0, right: true},
		{title: "role", min: 10, weight: 0},
		{title: "status", min: 10, weight: 0},
		{title: "qdb version", min: 14, weight: 0},
	})

	title := m.styles.label.Render("quarkdb cluster (qdb)")
	lines := []string{
		title,
		"",
		m.renderSimpleHeaderRow(columns, []string{"host", "port", "role", "status", "qdb version"}),
	}

	if m.mgmsLoading && len(mgms) == 0 {
		lines = append(lines, "loading qdb info...")
	} else if m.mgmsErr != nil {
		lines = append(lines, m.styles.error.Render(m.mgmsErr.Error()))
	} else if len(mgms) == 0 {
		lines = append(lines, "(no qdb nodes found)")
	} else {
		for i, node := range mgms {
			row := formatTableRow(columns, []string{
				node.QDBHost,
				fmt.Sprintf("%d", node.QDBPort),
				strings.ToLower(node.Role),
				strings.ToLower(node.Status),
				node.EOSVersion,
			})
			if i == m.qdbSelected {
				row = m.styles.selected.Width(contentWidth).Render(row)
			}
			lines = append(lines, row)
		}
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, height))
}
