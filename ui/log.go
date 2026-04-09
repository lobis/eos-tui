package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// selectedHostForView returns the hostname of the currently selected row in the
// active view, or empty string when no row is selected or the view has no host.
func (m model) selectedHostForView() string {
	switch m.activeView {
	case viewMGM:
		if m.mgmSelected >= 0 && m.mgmSelected < len(m.mgms) {
			return m.mgms[m.mgmSelected].Host
		}
	case viewQDB:
		if m.qdbSelected >= 0 && m.qdbSelected < len(m.mgms) {
			return m.mgms[m.qdbSelected].QDBHost
		}
	case viewFST:
		fsts := m.visibleFSTs()
		if m.fstSelected >= 0 && m.fstSelected < len(fsts) {
			return fsts[m.fstSelected].Host
		}
	case viewFileSystems:
		fss := m.visibleFileSystems()
		if m.fsSelected >= 0 && m.fsSelected < len(fss) {
			return fss[m.fsSelected].Host
		}
	}
	return ""
}

// logFileForView returns the log file path and a human-readable title for the
// currently active view.
func (m model) logFileForView() (filePath, title string) {
	switch m.activeView {
	case viewQDB:
		return "/var/log/eos/quarkdb/xrdlog.quarkdb", "QDB Log"
	case viewFST, viewFileSystems:
		return "/var/log/eos/fst/xrdlog.fst", "FST Log"
	default:
		return "/var/log/eos/mgm/xrdlog.mgm", "MGM Log"
	}
}

func (m model) openLogOverlay() (tea.Model, tea.Cmd) {
	if m.client == nil {
		return m, nil
	}
	filePath, title := m.logFileForView()
	host := m.selectedHostForView()

	logInput := textinput.New()
	logInput.Prompt = "grep> "
	logInput.CharLimit = 256

	vp := viewport.New(m.contentWidth()-4, max(4, m.height-10))
	vp.SetContent("Loading...")

	titleWithHost := title
	if host != "" {
		titleWithHost = fmt.Sprintf("%s  [%s]", title, host)
	}

	m.log = logOverlay{
		active:   true,
		host:     host,
		filePath: filePath,
		title:    titleWithHost,
		loading:  true,
		vp:       vp,
		input:    logInput,
	}
	return m, tea.Batch(loadLogCmd(m.client, host, filePath), logTickCmd())
}

func (m model) renderLogOverlay(height int) string {
	width := m.contentWidth()
	vpWidth := width - 4 // panel border + padding

	// Keep viewport sized to available space.
	filterHeight := 0
	if m.log.filtering {
		filterHeight = 2
	}
	vpHeight := max(4, height-4-filterHeight) // title bar (2) + border (2)
	m.log.vp.Width = vpWidth
	m.log.vp.Height = vpHeight

	// Title bar.
	filterInfo := ""
	if m.log.filter != "" {
		filterInfo = fmt.Sprintf("  [grep: %q  %d lines]", m.log.filter, len(m.log.filtered))
	}
	totalInfo := fmt.Sprintf("  %d lines", len(m.log.allLines))
	if m.log.loading {
		totalInfo = "  loading..."
	} else if m.log.err != nil {
		totalInfo = "  " + m.log.err.Error()
	}
	titleLine := m.styles.header.Render(m.log.title) +
		m.styles.label.Render("  "+m.log.filePath) +
		m.styles.value.Render(totalInfo+filterInfo)

	lines := []string{titleLine, ""}

	if m.log.err != nil && !m.log.loading {
		lines = append(lines, m.styles.error.Render(m.log.err.Error()))
	} else {
		lines = append(lines, m.log.vp.View())
	}

	if m.log.filtering {
		lines = append(lines, "", m.log.input.View())
	}

	inner := strings.Join(lines, "\n")
	return m.styles.panel.Width(width).Render(inner)
}

// applyLogFilter returns lines that case-insensitively contain filter.
// Empty filter returns all lines.
func applyLogFilter(lines []string, filter string) []string {
	if filter == "" {
		return lines
	}
	lower := strings.ToLower(filter)
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if strings.Contains(strings.ToLower(l), lower) {
			out = append(out, l)
		}
	}
	return out
}
