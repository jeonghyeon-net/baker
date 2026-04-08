package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type textInputModel struct {
	title       string
	hint        string
	input       textinput.Model
	value       string
	submitted   bool
	placeholder string
	height      int
}

func NewTextInputModel(title, hint, placeholder string) textInputModel {
	input := textinput.New()
	input.Placeholder = placeholder
	input.Focus()
	input.CharLimit = 512
	input.Width = 80

	return textInputModel{
		title:       title,
		hint:        hint,
		input:       input,
		placeholder: placeholder,
	}
}

func (m textInputModel) Init() tea.Cmd { return textinput.Blink }

func (m textInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.input.Width = msg.Width - 4
		if m.input.Width < 20 {
			m.input.Width = 20
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			m.value = m.input.Value()
			m.submitted = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m textInputModel) View() string {
	body := m.input.View()
	content := panelStyle.Render(renderPanel(withDefaultTitle(m.title, "Input"), "Text input", body, withDefaultHint(m.hint, "enter submit • esc cancel")))
	return appStyle.Render(content)
}

func PromptText(title, hint, placeholder string) (string, error) {
	finalModel, err := tea.NewProgram(NewTextInputModel(title, hint, placeholder), tea.WithAltScreen()).Run()
	if err != nil {
		return "", err
	}
	model, ok := finalModel.(textInputModel)
	if !ok {
		return "", nil
	}
	if !model.submitted {
		return "", nil
	}
	return model.value, nil
}
