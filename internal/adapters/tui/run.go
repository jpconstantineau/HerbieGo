package tui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

// Run launches the main Bubble Tea shell for the current match snapshot.
func Run(definition scenario.Definition, initial domain.MatchState) error {
	program := tea.NewProgram(
		NewModel(definition, newStaticStateSource(initial)),
		tea.WithAltScreen(),
	)
	_, err := program.Run()
	return err
}
