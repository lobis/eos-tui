package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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
	host := m.selectedHostForView()
	if host == "" {
		// Views without an associated host (namespace, spaces, stats, …) do not
		// support log tailing — 'l' is a no-op there.
		return m, nil
	}
	filePath, title := m.logFileForView()

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
		tailing:  true,
		loading:  true,
		vp:       vp,
		input:    logInput,
	}
	return m, tea.Batch(loadLogCmd(m.client, host, filePath), logTickCmd())
}

func (m model) logViewportWidth() int {
	width := m.contentWidth()
	if !m.log.plain {
		// In lipgloss v1, Width(w) includes padding but NOT borders.  We call
		// Width(contentWidth-2) so the outer panel = contentWidth.  The actual
		// inner content area = (contentWidth-2) - 2*padding = contentWidth - 4.
		width -= 4
	}
	return max(1, width)
}

func renderWrappedLogLines(lines []string, width int, wrap bool) []string {
	if width <= 0 {
		return append([]string(nil), lines...)
	}

	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if wrap {
			wrapped := ansi.Hardwrap(line, width, true)
			out = append(out, strings.Split(wrapped, "\n")...)
		} else {
			// Truncate so long lines never overflow the panel and eat the right border.
			out = append(out, ansi.Truncate(line, width, ""))
		}
	}
	return out
}

func (m *model) refreshLogViewportContent(preserveOffset bool) {
	width := m.logViewportWidth()
	wasAtBottom := m.log.vp.AtBottom()
	prevOffset := m.log.vp.YOffset
	m.log.vp.Width = width
	rendered := renderWrappedLogLines(m.log.filtered, width, m.log.wrap)
	m.log.vp.SetContent(strings.Join(rendered, "\n"))
	if !preserveOffset {
		return
	}
	if wasAtBottom {
		m.log.vp.GotoBottom()
		return
	}
	maxOffset := max(0, m.log.vp.TotalLineCount()-m.log.vp.Height)
	m.log.vp.SetYOffset(min(prevOffset, maxOffset))
}

func (m model) renderLogOverlay(height int) string {
	width := m.contentWidth()
	vpWidth := m.logViewportWidth()
	hasCachedContent := len(m.log.allLines) > 0

	// Keep viewport sized to available space.
	filterHeight := 0
	if m.log.filtering {
		filterHeight = 2
	}
	maxBoxViewportHeight := max(1, height-3-filterHeight) // title line + border (2) [+ filter input block]
	vpHeight := maxBoxViewportHeight
	if m.log.plain {
		vpHeight = max(4, height-filterHeight)
	} else if m.log.err != nil && !m.log.loading && !hasCachedContent {
		vpHeight = 1
	}
	// Always use the full available height so the panel fills the screen
	// even when only a few log lines have been loaded.
	m.log.vp.Width = vpWidth
	m.log.vp.Height = vpHeight
	m.log.vp.SetYOffset(m.log.vp.YOffset)

	if m.log.plain {
		lines := []string{m.renderLogViewport()}
		if m.log.filtering {
			lines = append(lines, "", m.log.input.View())
		}
		return strings.Join(lines, "\n")
	}

	// Title bar.
	filterInfo := ""
	if m.log.filter != "" {
		filterInfo = fmt.Sprintf("  [grep: %q  %d lines]", m.log.filter, len(m.log.filtered))
	}
	totalInfo := fmt.Sprintf("  %d lines", len(m.log.allLines))
	if m.log.loading {
		totalInfo = "  loading..."
	} else if m.log.err != nil {
		if hasCachedContent {
			totalInfo = "  reload failed; showing cached lines"
		} else {
			totalInfo = "  failed to load log"
		}
	}
	titleLine := m.styles.popupTitle.Render(m.log.title) +
		m.styles.label.Render("  "+m.log.filePath) +
		m.styles.value.Render(totalInfo+filterInfo)
	// Ensure the title line never overflows the inner panel width — if it does,
	// lipgloss v1 silently expands the panel box, shifting the right border off screen.
	titleLine = padVisibleWidth(titleLine, vpWidth)

	lines := []string{titleLine}

	if m.log.err != nil && !m.log.loading && !hasCachedContent {
		// Truncate error messages so they don't overflow the panel width.
		lines = append(lines, padVisibleWidth(m.styles.error.Render(ansi.Truncate(m.log.err.Error(), vpWidth, "…")), vpWidth))
	} else {
		lines = append(lines, m.renderLogViewport())
	}

	if m.log.filtering {
		lines = append(lines, "", m.log.input.View())
	}

	inner := strings.Join(lines, "\n")
	// In lipgloss v1, Width() sets content+padding width; borders are added on
	// top.  With Padding(0,1) and NormalBorder the border contributes 2 extra
	// chars (left+right), so the rendered outer width = Width + 2.  We want
	// outer = m.contentWidth() so that normalizeRenderedBlock doesn't clip the
	// right border, therefore pass Width(contentWidth - 2).
	panel := m.styles.panel.Width(width - 2).Render(inner)
	panelHeight := lipgloss.Height(panel)
	if panelHeight >= height {
		return panel
	}
	return strings.Repeat("\n", height-panelHeight) + panel
}

func (m model) renderLogViewport() string {
	w := m.logViewportWidth()
	lines := renderWrappedLogLines(m.log.filtered, w, m.log.wrap)
	top := max(0, min(m.log.vp.YOffset, len(lines)))
	bottom := min(top+m.log.vp.Height, len(lines))

	var visible []string
	if top < bottom {
		visible = lines[top:bottom]
	}

	// Prepend blank lines so content is bottom-aligned within the viewport.
	// This keeps the panel box full-height while the newest lines sit
	// naturally at the bottom, matching tail(1) behaviour.
	for len(visible) < m.log.vp.Height {
		visible = append([]string{""}, visible...)
	}

	// Explicitly pad every line to the viewport width so that lipgloss draws
	// the right border correctly even when lines are shorter than the panel.
	for i, line := range visible {
		visible[i] = padVisibleWidth(line, w)
	}

	return strings.Join(visible, "\n")
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
