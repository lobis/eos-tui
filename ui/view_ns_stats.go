package ui

import "fmt"

func (m model) renderNamespaceStatsView(height int) string {
	width := m.panelWidth()
	contentWidth := panelContentWidth(width)
	lines := []string{
		m.renderSectionTitle("General Statistics", contentWidth),
		"",
		m.renderSectionTitle("Cluster Summary", contentWidth),
	}
	switch {
	case m.fstStatsLoading:
		lines = append(lines, m.styles.value.Render("Loading cluster summary..."))
	case m.nodeStatsErr != nil:
		lines = append(lines, m.styles.error.Render(m.nodeStatsErr.Error()))
	default:
		lines = append(lines,
			m.metricLine("Health", fallback(m.nodeStats.State, "-"), "Threads", fmt.Sprintf("%d", m.nodeStats.ThreadCount)),
			m.metricLine("Files", fmt.Sprintf("%d", m.nodeStats.FileCount), "Dirs", fmt.Sprintf("%d", m.nodeStats.DirCount)),
			m.metricLine("Uptime", formatDuration(m.nodeStats.Uptime), "FDs", fmt.Sprintf("%d", m.nodeStats.FileDescs)),
		)
	}

	lines = append(lines, "", m.renderSectionTitle("Namespace Statistics", contentWidth))
	switch {
	case m.nsStatsLoading:
		lines = append(lines, m.styles.value.Render("Loading namespace statistics..."))
	case m.nsStatsErr != nil:
		lines = append(lines, m.styles.error.Render(m.nsStatsErr.Error()))
	default:
		stats := m.namespaceStats
		lines = append(lines,
			m.metricLine("Master", fallback(stats.MasterHost, "-"), "Total Files", fmt.Sprintf("%d", stats.TotalFiles)),
			m.metricLine("Total Directories", fmt.Sprintf("%d", stats.TotalDirectories), "", ""),
			"",
			m.renderSectionTitle("IDs", contentWidth),
			m.metricLine("Current File ID", fmt.Sprintf("%d", stats.CurrentFID), "Current Container ID", fmt.Sprintf("%d", stats.CurrentCID)),
			m.metricLine("Generated File IDs", fmt.Sprintf("%d", stats.GeneratedFID), "Generated Container IDs", fmt.Sprintf("%d", stats.GeneratedCID)),
			"",
			m.renderSectionTitle("Lock Contention", contentWidth),
			m.metricLine("Read Contention", fmt.Sprintf("%.2f", stats.ContentionRead), "Write Contention", fmt.Sprintf("%.2f", stats.ContentionWrite)),
			"",
			m.renderSectionTitle("File Cache", contentWidth),
			m.metricLine("Max Size", fmt.Sprintf("%d", stats.CacheFilesMax), "Occupancy", fmt.Sprintf("%d", stats.CacheFilesOccup)),
			m.metricLine("Requests", fmt.Sprintf("%d", stats.CacheFilesRequests), "Hits", fmt.Sprintf("%d", stats.CacheFilesHits)),
			"",
			m.renderSectionTitle("Container Cache", contentWidth),
			m.metricLine("Max Size", fmt.Sprintf("%d", stats.CacheContainersMax), "Occupancy", fmt.Sprintf("%d", stats.CacheContainersOccup)),
			m.metricLine("Requests", fmt.Sprintf("%d", stats.CacheContainersRequests), "Hits", fmt.Sprintf("%d", stats.CacheContainersHits)),
		)
	}

	return m.styles.panelDim.Width(width).Render(normalizePanelLines(lines, contentWidth, height))
}
