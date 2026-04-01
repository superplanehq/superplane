package widgets

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Prompt struct {
	input       textarea.Model
	writer      io.Writer
	width       int
	submitted   bool
	canceled    bool
	help        string
	placeholder string
}

func NewPrompt(writer io.Writer) *Prompt {
	input := textarea.New()
	input.Placeholder = "Ask SuperPlane to inspect code, plan edits, or continue the current chat..."
	input.ShowLineNumbers = false
	input.SetHeight(4)
	input.Prompt = "› "
	input.CharLimit = 0
	input.KeyMap.InsertNewline.SetEnabled(false)
	input.FocusedStyle.CursorLine = lipgloss.NewStyle()
	input.FocusedStyle.CursorLineNumber = lipgloss.NewStyle()
	input.FocusedStyle.Prompt = lipgloss.NewStyle().Bold(true)
	input.BlurredStyle = input.FocusedStyle
	input.FocusedStyle.Base = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		Padding(0, 1)
	input.Focus()

	return &Prompt{
		input:     input,
		writer:    writer,
		width:     0,
		submitted: false,
		canceled:  false,
		help:      "Enter send • Ctrl+J newline • Ctrl+C quit",
	}
}

func (p *Prompt) Init() tea.Cmd {
	return nil
}

func (p *Prompt) Run() (string, bool, error) {
	program := tea.NewProgram(p, tea.WithOutput(p.writer))
	result, err := program.Run()
	if err != nil {
		return "", false, err
	}

	finalModel, ok := result.(*Prompt)
	if !ok {
		return "", false, fmt.Errorf("unexpected prompt widget result")
	}

	clearRenderedLines(p.writer, lipgloss.Height(finalModel.View()))
	if finalModel.canceled {
		return "", true, nil
	}

	return strings.TrimSpace(finalModel.input.Value()), false, nil
}

func (p *Prompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = max(28, msg.Width)
		p.input.SetWidth(max(24, p.width-2))
		return p, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			p.canceled = true
			return p, tea.Quit
		case "enter":
			if strings.TrimSpace(p.input.Value()) == "" {
				return p, nil
			}
			p.submitted = true
			return p, tea.Quit
		case "ctrl+j", "alt+enter":
			p.input.InsertRune('\n')
			return p, nil
		}
	}

	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return p, cmd
}

func (p *Prompt) View() string {
	help := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "244", Dark: "241"}).
		Render(p.help)

	return lipgloss.JoinVertical(lipgloss.Left, p.input.View(), help)
}
