package widgets

import (
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Picker struct {
	writer  io.Writer
	options []PickerOption
	index   int
}

type PickerOption struct {
	ID       string
	Title    string
	Subtitle string
}

func NewPicker(writer io.Writer, options []PickerOption) *Picker {
	return &Picker{
		writer:  writer,
		options: options,
		index:   0,
	}
}

func (p *Picker) Run() (string, bool, error) {
	program := tea.NewProgram(p, tea.WithOutput(p.writer))
	result, err := program.Run()
	if err != nil {
		return "", false, err
	}

	finalModel, ok := result.(*Picker)
	if !ok {
		return "", false, fmt.Errorf("unexpected chat picker result")
	}

	clearRenderedLines(p.writer, lipgloss.Height(finalModel.View()))
	if finalModel.index < 0 || finalModel.index >= len(finalModel.options) {
		return "", true, nil
	}

	return finalModel.options[finalModel.index].ID, false, nil
}

func (p *Picker) Init() tea.Cmd { return nil }

func (p *Picker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			p.index = -1
			return p, tea.Quit
		case "up", "k", "shift+tab":
			if len(p.options) == 0 {
				return p, nil
			}
			p.index = (p.index + len(p.options) - 1) % len(p.options)
			return p, nil
		case "down", "j", "tab":
			if len(p.options) == 0 {
				return p, nil
			}
			p.index = (p.index + 1) % len(p.options)
			return p, nil
		case "enter":
			return p, tea.Quit
		}
	}

	return p, nil
}

func (p *Picker) View() string {
	title := lipgloss.NewStyle().Bold(true).Render("Resume chat")
	help := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "244", Dark: "241"}).
		Render("Up/Down move • Enter choose • Ctrl+C quit")

	selectedTitle := lipgloss.NewStyle().Bold(true)
	meta := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "244", Dark: "241"})

	rows := make([]string, 0, len(p.options))
	for i, option := range p.options {
		prefix := "  "
		line := option.Title

		if i == p.index {
			prefix = "› "
			line = selectedTitle.Render(option.Title)
		}

		if strings.TrimSpace(option.Subtitle) != "" {
			line += meta.Render("  " + option.Subtitle)
		}

		rows = append(rows, prefix+line)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.JoinVertical(lipgloss.Left, rows...),
		help,
	)
}
