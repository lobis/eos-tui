package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lobis/eos-tui/eos"
)

// ioShapingMergedRows returns the union of traffic records and policy records
// for the current mode, sorted alphabetically by id. Rows with traffic but no
// policy, policy but no traffic, or both are all included.
func (m model) ioShapingMergedRows() []ioShapingMergedRow {
	policyType := "app"
	switch m.ioShapingMode {
	case eos.IOShapingUsers:
		policyType = "uid"
	case eos.IOShapingGroups:
		policyType = "gid"
	}

	policyByID := make(map[string]eos.IOShapingPolicyRecord)
	for _, p := range m.ioShapingPolicies {
		if strings.ToLower(p.Type) == policyType {
			policyByID[p.ID] = p
		}
	}

	seen := make(map[string]bool)
	var rows []ioShapingMergedRow
	for i := range m.ioShaping {
		r := &m.ioShaping[i]
		seen[r.ID] = true
		row := ioShapingMergedRow{id: r.ID, traffic: r}
		if p, ok := policyByID[r.ID]; ok {
			row.policy = &p
		}
		rows = append(rows, row)
	}
	for id, p := range policyByID {
		if !seen[id] {
			pc := p
			rows = append(rows, ioShapingMergedRow{id: id, policy: &pc})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].id < rows[j].id })
	return rows
}

func (m model) renderIOShapingView(height int) string {
	width := m.contentWidth()
	contentWidth := panelContentWidth(width)

	idLabel := "application"
	switch m.ioShapingMode {
	case eos.IOShapingUsers:
		idLabel = "uid"
	case eos.IOShapingGroups:
		idLabel = "gid"
	}

	indicator := ""
	if m.ioShapingLoading {
		indicator = m.styles.status.Render("  ↻")
	}

	if m.ioShapingErr != nil {
		lines := []string{
			m.styles.label.Render("IO Traffic Shaping") + indicator,
			"",
			m.styles.error.Render(m.ioShapingErr.Error()),
		}
		return m.styles.panelDim.Width(width).Render(fitLines(lines, height))
	}

	rows := m.ioShapingMergedRows()

	formatLimit := func(v float64) string {
		if v == 0 {
			return "-"
		}
		return humanBytesRate(v)
	}
	enabledStr := func(p *eos.IOShapingPolicyRecord) string {
		if p == nil {
			return "-"
		}
		if p.Enabled {
			return "yes"
		}
		return "no"
	}

	dataRows := make([][]string, len(rows))
	for i, r := range rows {
		readRate, writeRate, readIOPS, writeIOPS := "-", "-", "-", "-"
		if r.traffic != nil {
			readRate = humanBytesRate(r.traffic.ReadBps)
			writeRate = humanBytesRate(r.traffic.WriteBps)
			readIOPS = fmt.Sprintf("%.1f", r.traffic.ReadIOPS)
			writeIOPS = fmt.Sprintf("%.1f", r.traffic.WriteIOPS)
		}
		limRead, limWrite, resRead, resWrite := "-", "-", "-", "-"
		if r.policy != nil {
			limRead = formatLimit(r.policy.LimitReadBytesPerSec)
			limWrite = formatLimit(r.policy.LimitWriteBytesPerSec)
			resRead = formatLimit(r.policy.ReservationReadBytesPerSec)
			resWrite = formatLimit(r.policy.ReservationWriteBytesPerSec)
		}
		dataRows[i] = []string{
			r.id,
			readRate, writeRate, readIOPS, writeIOPS,
			enabledStr(r.policy),
			limRead, limWrite, resRead, resWrite,
		}
	}

	headers := []string{idLabel, "read rate", "write rate", "read iops", "write iops", "enabled", "lim read", "lim write", "res read", "res write"}
	columns := allocateTableColumns(contentWidth, contentAwareColumns([]tableColumn{
		{title: idLabel, min: 10, weight: 4},
		{title: "read rate", min: 10, weight: 1, right: true},
		{title: "write rate", min: 10, weight: 1, right: true},
		{title: "read iops", min: 9, weight: 0, right: true},
		{title: "write iops", min: 10, weight: 0, right: true},
		{title: "enabled", min: 7, weight: 0},
		{title: "lim read", min: 10, weight: 0, right: true},
		{title: "lim write", min: 10, weight: 0, right: true},
		{title: "res read", min: 10, weight: 0, right: true},
		{title: "res write", min: 10, weight: 0, right: true},
	}, dataRows))

	title := m.styles.label.Render("IO Traffic  ") +
		m.styles.label.Render("5s window  ") +
		modeTabLabel(m.ioShapingMode, eos.IOShapingApps, "a apps", m.styles) + "  " +
		modeTabLabel(m.ioShapingMode, eos.IOShapingUsers, "u users", m.styles) + "  " +
		modeTabLabel(m.ioShapingMode, eos.IOShapingGroups, "g groups", m.styles) +
		indicator

	lines := []string{title, "", m.renderSimpleHeaderRow(columns, headers)}

	if m.ioShapingLoading && len(rows) == 0 {
		lines = append(lines, "Loading...")
	} else if len(rows) == 0 {
		lines = append(lines, "(no data)")
	} else {
		start, end := visibleWindow(len(rows), m.ioShapingSelected, max(1, height-len(lines)))
		lines[0] = title + renderScrollSummary(start, end, len(rows))
		for i := start; i < end; i++ {
			line := formatTableRow(columns, dataRows[i])
			if i == m.ioShapingSelected {
				line = m.styles.selected.Width(contentWidth).Render(line)
			}
			lines = append(lines, line)
		}
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, height))
}

func modeTabLabel(current, target eos.IOShapingMode, label string, s styles) string {
	if current == target {
		return s.tabActive.Render(label)
	}
	return s.tab.Render(label)
}

func humanBytesRate(bps float64) string {
	switch {
	case bps >= 1e9:
		return fmt.Sprintf("%.2f GB/s", bps/1e9)
	case bps >= 1e6:
		return fmt.Sprintf("%.2f MB/s", bps/1e6)
	case bps >= 1e3:
		return fmt.Sprintf("%.2f KB/s", bps/1e3)
	default:
		return fmt.Sprintf("%.0f B/s", bps)
	}
}
