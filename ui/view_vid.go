package ui

import (
	"github.com/lobis/eos-tui/eos"
)

func (m model) renderVIDView(height int) string {
	width := m.panelWidth()
	return m.renderVIDList(width, height)
}

func (m model) renderVIDList(width, height int) string {
	contentWidth := panelContentWidth(width)

	entries := m.visibleVID()
	dataRows := make([][]string, len(entries))
	for i, v := range entries {
		dataRows[i] = []string{v.Auth, v.Match, v.UID, v.GID}
	}

	columnDefs := contentAwareColumns([]tableColumn{
		{title: "auth", min: 8, weight: 1},
		{title: "match", min: 20, weight: 4},
		{title: "uid", min: 6, weight: 0, right: true},
		{title: "gid", min: 6, weight: 0, right: true},
	}, dataRows)

	columns := allocateTableColumns(contentWidth, columnDefs)

	title := m.renderSectionTitle("EOS VID Mappings", contentWidth)
	lines := []string{
		title,
		"",
		m.renderVIDHeaderRow(columns),
	}

	if m.vidLoading {
		lines = append(lines, "Loading VID mappings...")
	} else if m.vidErr != nil {
		lines = append(lines, m.styles.error.Render(m.vidErr.Error()))
	} else if len(entries) == 0 {
		lines = append(lines, "(no VID mappings)")
	} else {
		start, end := visibleWindow(len(entries), m.vidSelected, max(1, panelContentHeight(height)-len(lines)))
		lines[0] = title + renderScrollSummary(start, end, len(entries))
		for i := start; i < end; i++ {
			v := entries[i]
			row := []string{v.Auth, v.Match, v.UID, v.GID}
			line := formatTableRow(columns, row)
			if i == m.vidSelected {
				line = m.styles.selected.Width(contentWidth).Render(line)
			}
			lines = append(lines, line)
		}
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, panelContentHeight(height)))
}

func (m model) renderVIDHeaderRow(columns []tableColumn) string {
	labels := []string{"auth", "match", "uid", "gid"}
	return m.renderSelectableHeaderRow(columns, labels, m.vidColumnSelected, sortState{column: -1}, m.vidFilter)
}

// selectedVID returns the currently highlighted VID record, if any.
func (m model) selectedVID() (eos.VIDRecord, bool) {
	entries := m.visibleVID()
	if len(entries) == 0 || m.vidSelected < 0 || m.vidSelected >= len(entries) {
		return eos.VIDRecord{}, false
	}
	return entries[m.vidSelected], true
}
