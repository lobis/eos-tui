package ui

import (
	"fmt"
	"strings"
)

type topologyHostKind int

const (
	topologyHostMGM topologyHostKind = iota
	topologyHostQDB
)

type topologyHostRow struct {
	kind    topologyHostKind
	host    string
	port    int
	role    string
	status  string
	version string
}

// renderMGMView shows the unified topology as two stacked tables:
// management nodes first, then the matching QDB hosts.
func (m model) renderMGMView(height int) string {
	width := m.panelWidth()
	contentWidth := panelContentWidth(width)
	mgmRows := m.topologyMGMRows()
	qdbRows := m.topologyQDBRows()

	title := m.styles.label.Render("management & quarkdb topology")
	lines := []string{title, ""}

	if m.mgmsLoading && len(m.mgms) == 0 {
		lines = append(lines, "loading management and quarkdb info...")
	} else if m.mgmsErr != nil {
		lines = append(lines, m.styles.error.Render(m.mgmsErr.Error()))
	} else if len(mgmRows) == 0 && len(qdbRows) == 0 {
		lines = append(lines, "(no management nodes found)")
	} else {
		availableRows := max(2, panelContentHeight(height)-7)
		topRows := max(1, availableRows/2)
		bottomRows := max(1, availableRows-topRows)
		selectedKind, selectedLocal, _ := m.selectedTopologyHostRow()

		lines = append(lines, m.renderSectionTitle("Management Nodes (MGM)", contentWidth))
		lines = append(lines, m.renderTopologyRows(contentWidth, topRows, mgmRows, selectedKind, selectedLocal, topologyHostMGM)...)
		lines = append(lines, "")
		lines = append(lines, m.renderSectionTitle("QuarkDB Hosts (QDB)", contentWidth))
		lines = append(lines, m.renderTopologyRows(contentWidth, bottomRows, qdbRows, selectedKind, selectedLocal, topologyHostQDB)...)
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, height))
}

func (m model) topologyMGMRows() []topologyHostRow {
	rows := make([]topologyHostRow, 0, len(m.mgms))
	for _, node := range m.mgms {
		if node.Host == "" {
			continue
		}
		version := node.EOSVersion
		if version == "" {
			version = m.eosVersion
		}
		rows = append(rows, topologyHostRow{
			kind:    topologyHostMGM,
			host:    node.Host,
			port:    node.Port,
			role:    strings.ToLower(node.Role),
			status:  strings.ToLower(node.Status),
			version: version,
		})
	}
	return rows
}

func (m model) topologyQDBRows() []topologyHostRow {
	rows := make([]topologyHostRow, 0, len(m.mgms))
	for _, node := range m.mgms {
		if node.QDBHost == "" {
			continue
		}
		version := node.QDBVersion
		if version == "" {
			version = node.EOSVersion
		}
		rows = append(rows, topologyHostRow{
			kind:    topologyHostQDB,
			host:    node.QDBHost,
			port:    node.QDBPort,
			role:    strings.ToLower(node.Role),
			status:  strings.ToLower(node.Status),
			version: version,
		})
	}
	return rows
}

func (m model) topologySelectableRows() []topologyHostRow {
	rows := append([]topologyHostRow{}, m.topologyMGMRows()...)
	rows = append(rows, m.topologyQDBRows()...)
	return rows
}

func (m model) selectedTopologyHostRow() (topologyHostKind, int, bool) {
	rows := m.topologySelectableRows()
	if m.mgmSelected < 0 || m.mgmSelected >= len(rows) {
		return topologyHostMGM, -1, false
	}
	selected := rows[m.mgmSelected]
	local := 0
	switch selected.kind {
	case topologyHostMGM:
		for i, row := range m.topologyMGMRows() {
			if row.host == selected.host && row.port == selected.port {
				local = i
				break
			}
		}
	case topologyHostQDB:
		for i, row := range m.topologyQDBRows() {
			if row.host == selected.host && row.port == selected.port {
				local = i
				break
			}
		}
	}
	return selected.kind, local, true
}

func (m model) renderTopologyRows(contentWidth, maxRows int, rows []topologyHostRow, selectedKind topologyHostKind, selectedLocal int, sectionKind topologyHostKind) []string {
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

	sectionSelected := -1
	if selectedKind == sectionKind {
		sectionSelected = selectedLocal
	}
	if len(rows) == 0 {
		return append(lines, "(none)")
	}

	for _, idx := range visibleTableIndices(len(rows), sectionSelected, maxRows) {
		rowData := rows[idx]
		row := formatTableRow(columns, []string{
			rowData.host,
			fmt.Sprintf("%d", rowData.port),
			rowData.role,
			rowData.status,
			rowData.version,
		})
		if idx == sectionSelected {
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

	if selected < 0 {
		selected = 0
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
