package ui

import "fmt"

func (m model) renderNamespaceStatsView(height int) string {
	width := m.contentWidth()
	contentWidth := panelContentWidth(width)

	if m.nsStatsLoading {
		return m.styles.panelDim.Width(width).Render(fitLines([]string{"Loading namespace statistics..."}, height))
	}
	if m.nsStatsErr != nil {
		return m.styles.panelDim.Width(width).Render(fitLines([]string{m.styles.error.Render(m.nsStatsErr.Error())}, height))
	}

	stats := m.namespaceStats
	lines := []string{
		m.styles.label.Render("Namespace Statistics"),
		"",
		m.metricLine("Total Files", fmt.Sprintf("%d", stats.TotalFiles), "Total Directories", fmt.Sprintf("%d", stats.TotalDirectories)),
		"",
		m.styles.label.Render("IDs"),
		m.metricLine("Current File ID", fmt.Sprintf("%d", stats.CurrentFID), "Current Container ID", fmt.Sprintf("%d", stats.CurrentCID)),
		m.metricLine("Generated File IDs", fmt.Sprintf("%d", stats.GeneratedFID), "Generated Container IDs", fmt.Sprintf("%d", stats.GeneratedCID)),
		"",
		m.styles.label.Render("Lock Contention"),
		m.metricLine("Read Contention", fmt.Sprintf("%.2f", stats.ContentionRead), "Write Contention", fmt.Sprintf("%.2f", stats.ContentionWrite)),
		"",
		m.styles.label.Render("File Cache"),
		m.metricLine("Max Size", fmt.Sprintf("%d", stats.CacheFilesMax), "Occupancy", fmt.Sprintf("%d", stats.CacheFilesOccup)),
		m.metricLine("Requests", fmt.Sprintf("%d", stats.CacheFilesRequests), "Hits", fmt.Sprintf("%d", stats.CacheFilesHits)),
		"",
		m.styles.label.Render("Container Cache"),
		m.metricLine("Max Size", fmt.Sprintf("%d", stats.CacheContainersMax), "Occupancy", fmt.Sprintf("%d", stats.CacheContainersOccup)),
		m.metricLine("Requests", fmt.Sprintf("%d", stats.CacheContainersRequests), "Hits", fmt.Sprintf("%d", stats.CacheContainersHits)),
	}

	return m.styles.panelDim.Width(contentWidth).Render(fitLines(lines, height))
}
