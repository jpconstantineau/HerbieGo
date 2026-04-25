package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

// Run launches the main Bubble Tea shell for the current match snapshot.
func Run(definition scenario.Definition, initial domain.MatchState) error {
	return RunWithSource(definition, newStaticStateSource(initial), nil, tea.WithAltScreen())
}

// NewProgram constructs the Bubble Tea program for the supplied match source.
func NewProgram(definition scenario.Definition, source StateSource, submit SubmitFunc, debug DebugSource, options ...tea.ProgramOption) *tea.Program {
	opts := append([]tea.ProgramOption{tea.WithMouseCellMotion()}, options...)
	model := NewModelWithSubmit(definition, source, submit)
	model.debugLog = debug
	return tea.NewProgram(model, opts...)
}

// RunWithSource launches the Bubble Tea shell for a live or static match source.
func RunWithSource(definition scenario.Definition, source StateSource, submit SubmitFunc, options ...tea.ProgramOption) error {
	program := NewProgram(definition, source, submit, nil, options...)
	_, err := program.Run()
	return err
}
