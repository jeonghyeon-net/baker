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
	return renderScreen(withDefaultTitle(m.title, "Input"), body, m.hint)
}

func PromptText(title, hint, placeholder string) (string, error) {
	finalModel, err := tea.NewProgram(NewTextInputModel(title, hint, placeholder)).Run()
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
