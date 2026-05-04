package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var splashTotalDuration = 3 * time.Second

const (
	splashCanvasWidth  = 72
	splashCanvasHeight = 14
)

type splashTickMsg struct{}

type splashModel struct {
	frame    int
	finished bool
	width    int
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
	subtitle := lipgloss.NewStyle().Foreground(lipgloss.Color("223"))
	frame := herbieSplashFrames()[min(m.frame, len(herbieSplashFrames())-1)]

	lines := []string{
		subtitle.Render("The line is moving."),
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
	return tea.Tick(splashFrameDelayForFrameCount(len(herbieSplashFrames())), func(time.Time) tea.Msg {
		return splashTickMsg{}
	})
}

func splashFrameDelayForFrameCount(frameCount int) time.Duration {
	if frameCount <= 0 {
		return splashTotalDuration
	}
	return splashTotalDuration / time.Duration(frameCount)
}

func herbieSplashFrames() []string {
	return []string{
		renderHerbieSplashFrame(-10),
		renderHerbieSplashFrame(-4),
		renderHerbieSplashFrame(2),
		renderHerbieSplashFrame(8),
		renderHerbieSplashFrame(14),
		renderHerbieSplashFrame(20),
	}
}

func renderHerbieSplashFrame(offset int) string {
	canvas := newSplashCanvas(splashCanvasWidth, splashCanvasHeight)

	canvas.write(4, 2, "    /---\\                                /---\\")
	canvas.write(4, 3, "   /     \\______________________________/     \\")
	canvas.write(4, 4, "  |   __        __                        []   |")
	canvas.write(4, 5, "  |  /  \\______/  \\                      []    |")
	canvas.write(4, 6, "  | |   o      o   |________________________   |")
	canvas.write(4, 7, "  | |      __      |========================\\  |")
	canvas.write(4, 8, "  | |     /  \\     |                         | |")
	canvas.write(4, 9, "  | |____/____\\____|                         | |")
	canvas.write(4, 10, "  |    ~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^~    | |")
	canvas.write(4, 11, "  |__________________________________________| |")
	canvas.write(0, 12, "===="+strings.Repeat("#=", 30)+"====")
	canvas.write(0, 13, "----"+strings.Repeat("#=", 30)+"----")

	canvas.write(14, 7, renderBeltWordmark(offset, 24))
	canvas.write(7, 13, renderLowerBeltEcho(offset, 16))

	return styleSplashCanvas(canvas.lines())
}

func renderBeltWordmark(offset, width int) string {
	base := repeatPattern("=x", width)
	word := "HERBIEGO"
	runes := []rune(base)
	for index := 0; index < width; index++ {
		wordIndex := index - offset
		if wordIndex >= 0 && wordIndex < len(word) {
			runes[index] = rune(word[wordIndex])
		}
	}
	return string(runes)
}

func renderLowerBeltEcho(offset, width int) string {
	base := repeatPattern("#=", width)
	word := strings.ToLower("HERBIEGO")
	runes := []rune(base)
	for index := 0; index < width; index++ {
		wordIndex := index - offset
		if wordIndex >= 0 && wordIndex < len(word) {
			runes[index] = rune(word[wordIndex])
		}
	}
	return string(runes)
}

func styleSplashCanvas(lines []string) string {
	steel := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	brick := lipgloss.NewStyle().Foreground(lipgloss.Color("209"))
	fire := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	word := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	heatedWord := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("228"))
	eyes := lipgloss.NewStyle().Foreground(lipgloss.Color("226"))

	styled := make([]string, 0, len(lines))
	for index, line := range lines {
		current := line
		current = strings.ReplaceAll(current, "HERB", word.Render("HERB"))
		current = strings.ReplaceAll(current, "IEGO", heatedWord.Render("IEGO"))
		current = strings.ReplaceAll(current, "herbiego", word.Render("herbiego"))
		current = strings.ReplaceAll(current, "o o", eyes.Render("o o"))
		current = strings.ReplaceAll(current, "~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^~", fire.Render("~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^~"))

		switch index {
		case 12, 13:
			current = steel.Render(current)
		default:
			current = brick.Render(current)
		}
		styled = append(styled, current)
	}
	return strings.Join(styled, "\n")
}

type splashCanvas struct {
	cells [][]rune
}

func newSplashCanvas(width, height int) splashCanvas {
	cells := make([][]rune, height)
	for row := range cells {
		cells[row] = make([]rune, width)
		for col := range cells[row] {
			cells[row][col] = ' '
		}
	}
	return splashCanvas{cells: cells}
}

func (c splashCanvas) write(col, row int, text string) {
	if row < 0 || row >= len(c.cells) {
		return
	}
	for index, char := range text {
		target := col + index
		if target < 0 || target >= len(c.cells[row]) {
			continue
		}
		c.cells[row][target] = char
	}
}

func (c splashCanvas) lines() []string {
	lines := make([]string, len(c.cells))
	for row := range c.cells {
		lines[row] = strings.TrimRight(string(c.cells[row]), " ")
	}
	return lines
}

func repeatPattern(pattern string, width int) string {
	if width <= 0 || pattern == "" {
		return ""
	}
	var builder strings.Builder
	for builder.Len() < width {
		builder.WriteString(pattern)
	}
	return builder.String()[:width]
}

func splashFrameVisibleWidths() []int {
	frames := herbieSplashFrames()
	widths := make([]int, len(frames))
	for index, frame := range frames {
		lines := strings.Split(frame, "\n")
		maxWidth := 0
		for _, line := range lines {
			if width := ansi.StringWidth(line); width > maxWidth {
				maxWidth = width
			}
		}
		widths[index] = maxWidth
	}
	return widths
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
