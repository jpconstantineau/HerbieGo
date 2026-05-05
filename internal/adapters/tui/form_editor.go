package tui

import (
	"strings"

	"github.com/jpconstantineau/herbiego/internal/actionschema"
)

type formFieldValue struct {
	Scalar string
	Rows   []map[string]string
}

type actionFormModel struct {
	Schema      actionschema.RoleSchema
	Values      map[string]formFieldValue
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
	}
	for _, field := range schema.Fields {
		model.Values[field.ID] = formFieldValue{}
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
	} else {
		value.Scalar = strings.TrimSpace(m.InputBuffer)
	}
	m.storeCurrentValue(value)
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
}

func cycleOptionValue(options []actionschema.Option, current string, delta int) (string, bool) {
	if len(options) == 0 {
		return "", false
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
