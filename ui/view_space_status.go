package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lobis/eos-tui/eos"
)

func (m model) renderSpaceStatusView(height int) string {
	width := m.panelWidth()
	contentWidth := panelContentWidth(width)
	spaceName := m.currentSpaceStatusName()

	if m.spaceStatusLoading {
		return m.styles.panelDim.Width(width).Render(fitLines([]string{fmt.Sprintf("Loading space status for %s...", spaceName)}, height))
	}
	if m.spaceStatusErr != nil {
		return m.styles.panelDim.Width(width).Render(fitLines([]string{m.styles.error.Render(m.spaceStatusErr.Error())}, height))
	}

	columns := allocateTableColumns(contentWidth, []tableColumn{
		{title: "parameter", min: 36, weight: 1},
		{title: "value", min: 40, weight: 2},
	})

	title := m.styles.label.Render(fmt.Sprintf("EOS Space Status (%s)", spaceName))
	lines := []string{
		title,
		"",
		m.renderSimpleHeaderRow(columns, []string{"parameter", "value"}),
	}

	if len(m.spaceStatus) == 0 {
		lines = append(lines, "(no space status entries)")
	} else {
		start, end := visibleWindow(len(m.spaceStatus), m.spaceStatusSelected, max(1, height-len(lines)))
		lines[0] = title + renderScrollSummary(start, end, len(m.spaceStatus))
		for i := start; i < end; i++ {
			record := m.spaceStatus[i]
			line := formatTableRow(columns, []string{record.Key, record.Value})
			if i == m.spaceStatusSelected {
				line = m.styles.selected.Width(contentWidth).Render(line)
			}
			lines = append(lines, line)
		}
	}

	return m.styles.panel.Width(width).Render(fitLines(lines, height))
}

func (m model) selectedSpaceStatusRecord() (eos.SpaceStatusRecord, bool) {
	if len(m.spaceStatus) == 0 || m.spaceStatusSelected < 0 || m.spaceStatusSelected >= len(m.spaceStatus) {
		return eos.SpaceStatusRecord{}, false
	}
	return m.spaceStatus[m.spaceStatusSelected], true
}

func (m model) startSpaceStatusEdit() (tea.Model, tea.Cmd) {
	record, ok := m.selectedSpaceStatusRecord()
	if !ok {
		return m, nil
	}

	input := textinput.New()
	input.Placeholder = "new value"
	input.Focus()
	input.SetValue(record.Value)

	m.edit = spaceStatusEdit{
		active:     true,
		stage:      editStageInput,
		space:      m.currentSpaceStatusName(),
		record:     record,
		input:      input,
		button:     buttonCancel,
		focusInput: true,
	}

	return m, nil
}

func (m model) renderSpaceStatusEditPopup() string {
	cancelBtn := "[ Cancel ]"
	continueBtn := "[ Continue ]"

	if !m.edit.focusInput {
		if m.edit.button == buttonCancel {
			cancelBtn = m.styles.selected.Render(cancelBtn)
		} else {
			continueBtn = m.styles.selected.Render(continueBtn)
		}
	}

	lines := []string{
		m.styles.popupTitle.Render("Edit Space Status"),
		fmt.Sprintf("Key:   %s", m.styles.value.Render(m.edit.record.Key)),
		fmt.Sprintf("Value: %s", m.styles.value.Render(m.edit.record.Value)),
		"",
		m.edit.input.View(),
		"",
		lipgloss.JoinHorizontal(lipgloss.Left, cancelBtn, "  ", continueBtn),
		"",
		m.styles.status.Render("tab switch focus  •  g cancel  •  G continue  •  enter next"),
	}

	return m.styles.panel.
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m model) renderSpaceStatusConfirmPopup() string {
	cancelBtn := "[ Cancel ]"
	confirmBtn := "[ Confirm ]"

	if m.edit.button == buttonCancel {
		cancelBtn = m.styles.selected.Render(cancelBtn)
	} else {
		confirmBtn = m.styles.selected.Render(confirmBtn)
	}

	command := fmt.Sprintf("eos space config %s %s=%s", m.edit.space, m.edit.record.Key, m.edit.input.Value())

	lines := []string{
		m.styles.popupTitle.Render("Confirm Configuration Change"),
		"",
		"The following command will be executed:",
		"",
		m.styles.value.Render(command),
		"",
		lipgloss.JoinHorizontal(lipgloss.Left, cancelBtn, "  ", confirmBtn),
		"",
		m.styles.status.Render("g cancel  •  G confirm  •  enter apply"),
	}

	return m.styles.panel.
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m model) currentSpaceStatusName() string {
	if m.spaceStatusTarget != "" {
		return m.spaceStatusTarget
	}
	if space, ok := m.selectedSpace(); ok {
		return space.Name
	}
	return "space"
}

func (m model) openSelectedSpaceStatus() (tea.Model, tea.Cmd) {
	space, ok := m.selectedSpace()
	if !ok {
		return m, nil
	}

	targetChanged := m.spaceStatusTarget != space.Name
	m.spaceStatusActive = true
	m.spaceStatusTarget = space.Name
	if targetChanged {
		m.spaceStatus = nil
		m.spaceStatusErr = nil
		m.spaceStatusSelected = 0
	}

	if !targetChanged && !m.spaceStatusLoading && m.spaceStatusErr == nil && len(m.spaceStatus) > 0 {
		m.status = fmt.Sprintf("Viewing space status for %s", space.Name)
		return m, nil
	}

	return m.maybeLoadSpaceStatus(space.Name)
}
