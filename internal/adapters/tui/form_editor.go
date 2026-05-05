package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/jpconstantineau/herbiego/internal/actionschema"
)

type formFieldValue struct {
	Scalar string
	Rows   []map[string]string
}

type actionFormModel struct {
	Schema      actionschema.RoleSchema
	Values      map[string]formFieldValue
	Tables      map[string]table.Model
	FieldIndex  int
	RowIndex    int
	ColumnIndex int
	Editing     bool
	InputBuffer string
}

func newActionFormModel(schema actionschema.RoleSchema) actionFormModel {
	model := actionFormModel{
		Schema: schema,
		Values: make(map[string]formFieldValue, len(schema.Fields)),
		Tables: make(map[string]table.Model, len(schema.Fields)),
	}
	for _, field := range schema.Fields {
		model.Values[field.ID] = formFieldValue{}
		if field.Collection != nil {
			model.Tables[field.ID] = newCollectionTable(field)
		}
	}
	return model
}

func (m *actionFormModel) currentField() *actionschema.FieldSpec {
	if m == nil || len(m.Schema.Fields) == 0 {
		return nil
	}
	index := m.FieldIndex
	if index < 0 {
		index = 0
	}
	if index >= len(m.Schema.Fields) {
		index = len(m.Schema.Fields) - 1
	}
	return &m.Schema.Fields[index]
}

func (m *actionFormModel) currentColumn() *actionschema.ColumnSpec {
	field := m.currentField()
	if field == nil || field.Collection == nil || len(field.Collection.Columns) == 0 {
		return nil
	}
	index := m.ColumnIndex
	if index < 0 {
		index = 0
	}
	if index >= len(field.Collection.Columns) {
		index = len(field.Collection.Columns) - 1
	}
	return &field.Collection.Columns[index]
}

func (m *actionFormModel) currentValue() formFieldValue {
	field := m.currentField()
	if field == nil {
		return formFieldValue{}
	}
	value, ok := m.Values[field.ID]
	if !ok {
		return formFieldValue{}
	}
	return value
}

func (m *actionFormModel) storeCurrentValue(value formFieldValue) {
	field := m.currentField()
	if field == nil {
		return
	}
	m.Values[field.ID] = value
}

func (m *actionFormModel) MoveUp() {
	field := m.currentField()
	if field != nil && field.Collection != nil {
		if m.RowIndex > 0 {
			m.RowIndex--
			m.syncCurrentTable()
			return
		}
	}
	if m.FieldIndex > 0 {
		m.FieldIndex--
		m.resetFocusForField()
	}
}

func (m *actionFormModel) MoveDown() {
	field := m.currentField()
	if field != nil && field.Collection != nil {
		rows := m.currentValue().Rows
		if m.RowIndex < len(rows)-1 {
			m.RowIndex++
			m.syncCurrentTable()
			return
		}
	}
	if m.FieldIndex < len(m.Schema.Fields)-1 {
		m.FieldIndex++
		m.resetFocusForField()
	}
}

func (m *actionFormModel) MoveLeft() {
	field := m.currentField()
	if field == nil || field.Collection == nil {
		return
	}
	if m.ColumnIndex > 0 {
		m.ColumnIndex--
	}
}

func (m *actionFormModel) MoveRight() {
	field := m.currentField()
	if field == nil || field.Collection == nil {
		return
	}
	if m.ColumnIndex < len(field.Collection.Columns)-1 {
		m.ColumnIndex++
	}
}

func (m *actionFormModel) AddRow() {
	field := m.currentField()
	if field == nil || field.Collection == nil {
		return
	}

	value := m.currentValue()
	row := make(map[string]string, len(field.Collection.Columns))
	for _, column := range field.Collection.Columns {
		row[column.ID] = ""
	}
	value.Rows = append(value.Rows, row)
	m.storeCurrentValue(value)
	m.RowIndex = len(value.Rows) - 1
	m.ColumnIndex = 0
	m.syncCurrentTable()
}

func (m *actionFormModel) RemoveRow() bool {
	field := m.currentField()
	if field == nil || field.Collection == nil {
		return false
	}
	value := m.currentValue()
	if len(value.Rows) == 0 || m.RowIndex < 0 || m.RowIndex >= len(value.Rows) {
		return false
	}
	value.Rows = append(value.Rows[:m.RowIndex], value.Rows[m.RowIndex+1:]...)
	if m.RowIndex >= len(value.Rows) && m.RowIndex > 0 {
		m.RowIndex--
	}
	m.storeCurrentValue(value)
	m.syncCurrentTable()
	return true
}

func (m *actionFormModel) CycleChoice(delta int) bool {
	if delta == 0 {
		return false
	}
	field := m.currentField()
	if field == nil {
		return false
	}

	if field.Collection != nil {
		column := m.currentColumn()
		if column == nil || column.Kind != actionschema.ValueKindChoice {
			return false
		}
		value := m.currentValue()
		if len(value.Rows) == 0 {
			m.AddRow()
			value = m.currentValue()
		}
		row := value.Rows[m.RowIndex]
		options := column.Options.Options(row)
		next, ok := cycleOptionValue(options, row[column.ID], delta)
		if !ok {
			return false
		}
		row[column.ID] = next
		if column.ID == column.Options.DependencyFieldID {
			row[column.ID] = next
		}
		value.Rows[m.RowIndex] = row
		m.storeCurrentValue(value)
		m.syncCurrentTable()
		return true
	}

	if field.Kind != actionschema.ValueKindChoice {
		return false
	}
	value := m.currentValue()
	next, ok := cycleOptionValue(field.Options.Static, value.Scalar, delta)
	if !ok {
		return false
	}
	value.Scalar = next
	m.storeCurrentValue(value)
	return true
}

func (m *actionFormModel) BeginEdit() bool {
	field := m.currentField()
	if field == nil {
		return false
	}
	if field.Collection != nil {
		column := m.currentColumn()
		if column == nil {
			return false
		}
		if column.Kind == actionschema.ValueKindChoice {
			return m.CycleChoice(1)
		}
		value := m.currentValue()
		if len(value.Rows) == 0 {
			m.AddRow()
			value = m.currentValue()
		}
		m.InputBuffer = value.Rows[m.RowIndex][column.ID]
		m.Editing = true
		return true
	}

	if field.Kind == actionschema.ValueKindChoice {
		return m.CycleChoice(1)
	}
	m.InputBuffer = m.currentValue().Scalar
	m.Editing = true
	return true
}

func (m *actionFormModel) CommitEdit() bool {
	if !m.Editing {
		return false
	}
	field := m.currentField()
	if field == nil {
		return false
	}
	value := m.currentValue()
	if field.Collection != nil {
		column := m.currentColumn()
		if column == nil || len(value.Rows) == 0 {
			return false
		}
		value.Rows[m.RowIndex][column.ID] = strings.TrimSpace(m.InputBuffer)
		m.storeCurrentValue(value)
		m.syncCurrentTable()
	} else {
		value.Scalar = strings.TrimSpace(m.InputBuffer)
		m.storeCurrentValue(value)
	}
	m.Editing = false
	m.InputBuffer = ""
	return true
}

func (m *actionFormModel) CancelEdit() bool {
	if !m.Editing {
		return false
	}
	m.Editing = false
	m.InputBuffer = ""
	return true
}

func (m *actionFormModel) TypeRunes(input string) bool {
	if !m.Editing {
		return false
	}
	m.InputBuffer += input
	return true
}

func (m *actionFormModel) Backspace() bool {
	if !m.Editing || len(m.InputBuffer) == 0 {
		return false
	}
	m.InputBuffer = m.InputBuffer[:len(m.InputBuffer)-1]
	return true
}

func (m actionFormModel) displayScalar(field actionschema.FieldSpec) string {
	value := m.Values[field.ID].Scalar
	if m.Editing && m.currentField() != nil && m.currentField().ID == field.ID && field.Collection == nil {
		return renderInputCursor(m.InputBuffer)
	}
	if strings.TrimSpace(value) == "" {
		return field.Placeholder
	}
	if field.Kind == actionschema.ValueKindChoice {
		return optionLabel(field.Options.Static, value)
	}
	return value
}

func (m actionFormModel) displayCell(field actionschema.FieldSpec, rowIndex int, column actionschema.ColumnSpec) string {
	value := m.Values[field.ID]
	if rowIndex >= len(value.Rows) {
		return column.Placeholder
	}
	cell := value.Rows[rowIndex][column.ID]
	if m.Editing && m.currentField() != nil && m.currentField().ID == field.ID && m.RowIndex == rowIndex && m.currentColumn() != nil && m.currentColumn().ID == column.ID {
		return renderInputCursor(m.InputBuffer)
	}
	if strings.TrimSpace(cell) == "" {
		return column.Placeholder
	}
	if column.Kind == actionschema.ValueKindChoice {
		return optionLabel(column.Options.Options(value.Rows[rowIndex]), cell)
	}
	return cell
}

func renderInputCursor(value string) string {
	return value + "|"
}

func (m *actionFormModel) resetFocusForField() {
	m.RowIndex = 0
	m.ColumnIndex = 0
	m.Editing = false
	m.InputBuffer = ""
	m.syncCurrentTable()
}

func cycleOptionValue(options []actionschema.Option, current string, delta int) (string, bool) {
	if len(options) == 0 {
		return "", false
	}
	if strings.TrimSpace(current) == "" {
		if delta < 0 {
			return options[len(options)-1].Value, true
		}
		return options[0].Value, true
	}
	index := 0
	for i, option := range options {
		if option.Value == current {
			index = i
			break
		}
	}
	index = (index + delta + len(options)) % len(options)
	return options[index].Value, true
}

func optionLabel(options []actionschema.Option, value string) string {
	for _, option := range options {
		if option.Value == value {
			return option.Label
		}
	}
	return value
}

func newCollectionTable(field actionschema.FieldSpec) table.Model {
	columns := make([]table.Column, 0, len(field.Collection.Columns))
	for _, column := range field.Collection.Columns {
		columns = append(columns, table.Column{
			Title: column.Label,
			Width: defaultCollectionColumnWidth(column),
		})
	}
	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(nil),
		table.WithFocused(false),
		table.WithHeight(5),
	)
	tbl.SetStyles(collectionTableStyles())
	return tbl
}

func collectionTableStyles() table.Styles {
	styles := table.DefaultStyles()
	styles.Header = styles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	styles.Selected = styles.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	return styles
}

func inactiveCollectionTableStyles() table.Styles {
	styles := collectionTableStyles()
	styles.Selected = styles.Cell
	return styles
}

func defaultCollectionColumnWidth(column actionschema.ColumnSpec) int {
	width := len(column.Label) + 2
	if len(column.Placeholder)+2 > width {
		width = len(column.Placeholder) + 2
	}
	if column.Kind == actionschema.ValueKindChoice && width < 14 {
		width = 14
	}
	if column.Kind == actionschema.ValueKindInteger && width < 8 {
		width = 8
	}
	if width < 10 {
		width = 10
	}
	if width > 24 {
		width = 24
	}
	return width
}

func (m *actionFormModel) syncCurrentTable() {
	field := m.currentField()
	if field == nil || field.Collection == nil {
		return
	}
	m.syncTable(field.ID)
}

func (m *actionFormModel) syncTable(fieldID string) {
	field := m.fieldByID(fieldID)
	if field == nil || field.Collection == nil {
		return
	}
	tbl, ok := m.Tables[fieldID]
	if !ok {
		tbl = newCollectionTable(*field)
	}
	tbl.SetRows(m.tableRows(*field))
	if rowCount := len(m.Values[fieldID].Rows); rowCount > 0 {
		cursor := m.RowIndex
		if cursor >= rowCount {
			cursor = rowCount - 1
		}
		if cursor < 0 {
			cursor = 0
		}
		tbl.SetCursor(cursor)
	} else {
		tbl.SetCursor(0)
	}
	m.Tables[fieldID] = tbl
}

func (m actionFormModel) fieldByID(fieldID string) *actionschema.FieldSpec {
	for i := range m.Schema.Fields {
		if m.Schema.Fields[i].ID == fieldID {
			return &m.Schema.Fields[i]
		}
	}
	return nil
}

func (m actionFormModel) tableRows(field actionschema.FieldSpec) []table.Row {
	rows := m.Values[field.ID].Rows
	result := make([]table.Row, 0, len(rows))
	for rowIndex := range rows {
		row := make(table.Row, 0, len(field.Collection.Columns))
		for _, column := range field.Collection.Columns {
			cell := m.displayCell(field, rowIndex, column)
			row = append(row, cell)
		}
		result = append(result, row)
	}
	return result
}

func (m *actionFormModel) renderedCollectionTable(field actionschema.FieldSpec, width int, active bool) string {
	tbl, ok := m.Tables[field.ID]
	if !ok {
		tbl = newCollectionTable(field)
	}
	tbl.SetStyles(tblStylesForActiveState(active))
	if active {
		tbl.Focus()
	} else {
		tbl.Blur()
	}
	tbl.SetColumns(m.tableColumns(field, width, active))
	tbl.SetRows(m.tableRows(field))
	if rowCount := len(m.Values[field.ID].Rows); rowCount > 0 {
		cursor := m.RowIndex
		if cursor >= rowCount {
			cursor = rowCount - 1
		}
		if cursor < 0 {
			cursor = 0
		}
		tbl.SetCursor(cursor)
	}
	height := len(m.Values[field.ID].Rows) + 1
	if height < 3 {
		height = 3
	}
	if height > 8 {
		height = 8
	}
	tbl.SetHeight(height + collectionTableHeaderHeight(tbl.Columns(), tblStylesForActiveState(active)))
	if width > 0 {
		tbl.SetWidth(width)
	}
	m.Tables[field.ID] = tbl
	return tbl.View()
}

func (m actionFormModel) tableColumns(field actionschema.FieldSpec, width int, active bool) []table.Column {
	columnCount := len(field.Collection.Columns)
	if columnCount == 0 {
		return nil
	}
	available := width - (columnCount * 3)
	if available < columnCount*8 {
		available = columnCount * 8
	}
	baseWidth := available / columnCount
	if baseWidth < 8 {
		baseWidth = 8
	}
	columns := make([]table.Column, 0, columnCount)
	for index, column := range field.Collection.Columns {
		title := column.Label
		if active && index == m.ColumnIndex {
			title = "[" + title + "]"
		}
		colWidth := baseWidth
		if preferred := defaultCollectionColumnWidth(column); preferred > colWidth {
			colWidth = preferred
		}
		if colWidth > 24 {
			colWidth = 24
		}
		columns = append(columns, table.Column{
			Title: title,
			Width: colWidth,
		})
	}
	return columns
}

func (m actionFormModel) currentCollectionSummary(field actionschema.FieldSpec) string {
	if field.Collection == nil {
		return ""
	}
	rowCount := len(m.Values[field.ID].Rows)
	if rowCount == 0 {
		return field.Collection.EmptyText + " Press a to add a row."
	}
	column := m.currentColumn()
	columnLabel := ""
	if column != nil {
		columnLabel = column.Label
	}
	return fmt.Sprintf("%d row(s). Selected row %d%s", rowCount, m.RowIndex+1, currentColumnSuffix(columnLabel))
}

func currentColumnSuffix(label string) string {
	if strings.TrimSpace(label) == "" {
		return ""
	}
	return ", column " + label
}

func tblStylesForActiveState(active bool) table.Styles {
	if active {
		return collectionTableStyles()
	}
	return inactiveCollectionTableStyles()
}

func collectionTableHeaderHeight(columns []table.Column, styles table.Styles) int {
	headers := make([]string, 0, len(columns))
	for _, col := range columns {
		if col.Width <= 0 {
			continue
		}
		headers = append(headers, styles.Header.Width(col.Width).MaxWidth(col.Width).Inline(true).Render(col.Title))
	}
	return lipgloss.Height(lipgloss.JoinHorizontal(lipgloss.Top, headers...))
}
