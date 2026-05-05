package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const spreadsheetColumnGap = 3

type spreadsheetColumn struct {
	Title string
	Width int
}

type spreadsheetCell struct {
	Text        string
	Placeholder string
	Active      bool
}

type spreadsheetComponent struct {
	Columns []spreadsheetColumn
	Rows    [][]spreadsheetCell
}

func (s spreadsheetComponent) Render() string {
	if len(s.Columns) == 0 {
		return ""
	}

	header := s.renderHeader()
	rule := strings.Repeat("─", s.totalWidth())
	rows := s.renderRows()
	return strings.Join(append([]string{header, rule}, rows...), "\n")
}

func (s spreadsheetComponent) renderHeader() string {
	parts := make([]string, 0, len(s.Columns))
	for _, column := range s.Columns {
		parts = append(parts, lipgloss.NewStyle().
			Width(column.Width).
			MaxWidth(column.Width).
			Bold(false).
			Render(fitSpreadsheetText(column.Title, column.Width)))
	}
	return strings.Join(parts, strings.Repeat(" ", spreadsheetColumnGap))
}

func (s spreadsheetComponent) renderRows() []string {
	if len(s.Rows) == 0 {
		return []string{s.renderEmptyRow()}
	}
	lines := make([]string, 0, len(s.Rows))
	for _, row := range s.Rows {
		lines = append(lines, s.renderRow(row))
	}
	return lines
}

func (s spreadsheetComponent) renderEmptyRow() string {
	row := make([]spreadsheetCell, 0, len(s.Columns))
	for index := range s.Columns {
		row = append(row, spreadsheetCell{
			Text:        "",
			Placeholder: "",
			Active:      index == 0,
		})
	}
	return s.renderRow(row)
}

func (s spreadsheetComponent) renderRow(row []spreadsheetCell) string {
	parts := make([]string, 0, len(s.Columns))
	for index, column := range s.Columns {
		cell := spreadsheetCell{}
		if index < len(row) {
			cell = row[index]
		}
		parts = append(parts, renderSpreadsheetCell(cell, column.Width))
	}
	return strings.Join(parts, strings.Repeat(" ", spreadsheetColumnGap))
}

func (s spreadsheetComponent) totalWidth() int {
	total := 0
	for _, column := range s.Columns {
		total += column.Width
	}
	total += (len(s.Columns) - 1) * spreadsheetColumnGap
	return total
}

func renderSpreadsheetCell(cell spreadsheetCell, width int) string {
	text := cell.Text
	if strings.TrimSpace(text) == "" {
		text = cell.Placeholder
	}
	text = fitSpreadsheetText(text, width)
	style := lipgloss.NewStyle().
		Width(width).
		MaxWidth(width)
	if cell.Active {
		style = style.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57"))
	}
	return style.Render(text)
}

func fitSpreadsheetText(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(text) <= width {
		return text
	}
	runes := []rune(text)
	if width <= 3 {
		return string(runes[:min(width, len(runes))])
	}
	limit := min(len(runes), width-3)
	return string(runes[:limit]) + "..."
}
