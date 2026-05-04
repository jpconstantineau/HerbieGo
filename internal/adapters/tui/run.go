package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

// Run launches the main Bubble Tea shell for the current match snapshot.
func Run(definition ScenarioReader, initial domain.MatchState) error {
	return RunWithSource(definition, newStaticStateSource(initial), nil, tea.WithAltScreen())
}

// RunReplay launches the shell with persisted state snapshots and optional AI traces.
func RunReplay(definition ScenarioReader, current domain.MatchState, snapshots []domain.MatchState, debugRecords []ports.AICallRecord) error {
	source := newReplayStateSource(current, snapshots)
	program := NewProgram(definition, source, nil, newStaticDebugSource(debugRecords), tea.WithAltScreen())
	_, err := program.Run()
	return err
}

// NewProgramWithQuitBehavior constructs the Bubble Tea program for the supplied
// match source and q-key behavior.
func NewProgramWithQuitBehavior(definition ScenarioReader, source StateSource, submit SubmitFunc, debug DebugSource, quitBehavior QuitBehavior, options ...tea.ProgramOption) *tea.Program {
	opts := append([]tea.ProgramOption{tea.WithMouseCellMotion()}, options...)
	model := NewModelWithSubmitDebugAndQuitBehavior(definition, source, submit, debug, quitBehavior)
	return tea.NewProgram(model, opts...)
}

// NewModelWithSubmitDebugAndQuitBehavior constructs the gameplay model with
// live submission, debug data, and a configurable q-key behavior.
func NewModelWithSubmitDebugAndQuitBehavior(definition ScenarioReader, source StateSource, submit SubmitFunc, debug DebugSource, quitBehavior QuitBehavior) Model {
	model := NewModelWithSubmitAndQuitBehavior(definition, source, submit, quitBehavior)
	model.debugLog = debug
	return model
}

// NewProgram constructs the Bubble Tea program for the supplied match source.
func NewProgram(definition ScenarioReader, source StateSource, submit SubmitFunc, debug DebugSource, options ...tea.ProgramOption) *tea.Program {
	return NewProgramWithQuitBehavior(definition, source, submit, debug, QuitBehaviorQuitProgram, options...)
}

// RunWithSource launches the Bubble Tea shell for a live or static match source.
func RunWithSource(definition ScenarioReader, source StateSource, submit SubmitFunc, options ...tea.ProgramOption) error {
	program := NewProgram(definition, source, submit, nil, options...)
	_, err := program.Run()
	return err
}
