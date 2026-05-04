package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var splashFrameDelay = 220 * time.Millisecond

type splashTickMsg struct{}

type splashModel struct {
	frame    int
	finished bool
	width    int
}

func runSplashScreen() error {
	_, err := tea.NewProgram(splashModel{}, tea.WithAltScreen()).Run()
	return err
}

func (m splashModel) Init() tea.Cmd {
	return splashTickCmd()
}

func (m splashModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		return m, nil
	case tea.KeyMsg:
		m.finished = true
		return m, tea.Quit
	case splashTickMsg:
		if m.frame >= len(herbieSplashFrames())-1 {
			m.finished = true
			return m, tea.Quit
		}
		m.frame++
		return m, splashTickCmd()
	}

	return m, nil
}

func (m splashModel) View() string {
	accent := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))
	subtitle := lipgloss.NewStyle().Foreground(lipgloss.Color("223"))
	frame := herbieSplashFrames()[min(m.frame, len(herbieSplashFrames())-1)]

	lines := []string{
		accent.Render("HerbieGo"),
		subtitle.Render("Prairie heat. Tight cash. Bottlenecks everywhere."),
		"",
		frame,
		"",
		subtitle.Render("Launching start menu..."),
	}

	width := m.width
	if width <= 0 {
		width = 96
	}

	return lipgloss.NewStyle().
		Padding(1, 2).
		Width(width).
		Render(strings.Join(lines, "\n"))
}

func splashTickCmd() tea.Cmd {
	return tea.Tick(splashFrameDelay, func(time.Time) tea.Msg {
		return splashTickMsg{}
	})
}

func herbieSplashFrames() []string {
	brick := lipgloss.NewStyle().Foreground(lipgloss.Color("209")).Render
	fire := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render
	steel := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render
	heat := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render
	glow := lipgloss.NewStyle().Foreground(lipgloss.Color("228")).Render

	baseTop := brick("          __________________________________________")
	baseMid := brick("         /") + steel("                                        /|")
	baseBottom := brick("        /________________________________________/ |")
	chassis := brick("        |  ") + glow("HERBIE") + steel("   heat-treating oven") + brick("           | |")
	conveyor := steel("        |  rails: ==========>                   | |")
	feet := brick("        |_______________________________________|/") + "\n" +
		steel("           O                               O")

	return []string{
		strings.Join([]string{
			baseTop,
			baseMid,
			baseBottom,
			chassis,
			brick("        |  [##############]      ") + fire("(((())))") + brick("     | |"),
			brick("        |  [##############]      ") + fire("(((())))") + brick("     | |"),
			conveyor,
			feet,
		}, "\n"),
		strings.Join([]string{
			baseTop,
			baseMid,
			baseBottom,
			chassis,
			brick("        | /[##############]      ") + fire("(((())))") + brick("     | |"),
			brick("        |/ [##############]      ") + fire("(((())))") + brick("     | |"),
			conveyor,
			feet,
		}, "\n"),
		strings.Join([]string{
			baseTop,
			baseMid,
			baseBottom,
			chassis,
			brick("        /  [##############]      ") + heat("  * * * ") + brick("     | |"),
			brick("       /   [##############]      ") + fire("(((())))") + brick("     | |"),
			conveyor,
			feet,
		}, "\n"),
		strings.Join([]string{
			baseTop,
			baseMid,
			baseBottom,
			chassis,
			brick("      /    [##############]      ") + heat(" *  *  *") + brick("     | |"),
			brick("     /     [##############]      ") + fire("(((())))") + brick("     | |"),
			conveyor,
			feet,
		}, "\n"),
	}
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
