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

	if m.spaceStatusLoading {
		return m.styles.panelDim.Width(width).Render(fitLines([]string{"Loading space status..."}, height))
	}
	if m.spaceStatusErr != nil {
		return m.styles.panelDim.Width(width).Render(fitLines([]string{m.styles.error.Render(m.spaceStatusErr.Error())}, height))
	}

	columns := allocateTableColumns(contentWidth, []tableColumn{
		{title: "parameter", min: 36, weight: 1},
		{title: "value", min: 40, weight: 2},
	})

	title := m.styles.label.Render("EOS Space Status (default)")
	lines := []string{
		title,
		"",
		m.renderSimpleHeaderRow(columns, []string{"parameter", "value"}),
	}

	start, end := visibleWindow(len(m.spaceStatus), m.spaceStatusSelected, max(1, height-len(lines)))
	for i := start; i < end; i++ {
		record := m.spaceStatus[i]
		line := formatTableRow(columns, []string{record.Key, record.Value})
		if i == m.spaceStatusSelected {
			line = m.styles.selected.Width(contentWidth).Render(line)
		}
		lines = append(lines, line)
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
		m.styles.label.Render("Edit Space Status"),
		fmt.Sprintf("Key:   %s", m.styles.value.Render(m.edit.record.Key)),
		fmt.Sprintf("Value: %s", m.styles.value.Render(m.edit.record.Value)),
		"",
		m.edit.input.View(),
		"",
		lipgloss.JoinHorizontal(lipgloss.Left, cancelBtn, "  ", continueBtn),
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

	command := fmt.Sprintf("eos space config default %s=%s", m.edit.record.Key, m.edit.input.Value())

	lines := []string{
		m.styles.label.Render("Confirm Configuration Change"),
		"",
		"The following command will be executed:",
		"",
		m.styles.value.Render(command),
		"",
		lipgloss.JoinHorizontal(lipgloss.Left, cancelBtn, "  ", confirmBtn),
	}

	return m.styles.panel.
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}
