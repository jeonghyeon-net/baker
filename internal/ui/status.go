package ui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type statusResult struct {
	value any
	err   error
}

type statusMsg struct {
	result statusResult
}

type statusModel struct {
	title    string
	subtitle string
	message  string
	width    int
	height   int
	spinner  spinner.Model
	resultCh <-chan statusResult
	result   statusResult
}

func newStatusModel(title, subtitle, message string, resultCh <-chan statusResult) statusModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = indicatorStyle
	return statusModel{
		title:    title,
		subtitle: subtitle,
		message:  message,
		spinner:  s,
		resultCh: resultCh,
	}
}

func (m statusModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, waitStatus(m.resultCh))
}

func (m statusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case statusMsg:
		m.result = msg.result
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m statusModel) View() string {
	body := m.spinner.View() + "  " + worktreeStyle.Render(m.message)
	content := renderFrame(renderPanel(m.title, m.subtitle, body, metaStyle.Render("명령 실행 중에는 esc로 취소할 수 없습니다.")), m.width)
	return appStyle.Render(content)
}

func waitStatus(resultCh <-chan statusResult) tea.Cmd {
	return func() tea.Msg {
		return statusMsg{result: <-resultCh}
	}
}

func RunStatusValue[T any](title, subtitle, message string, fn func() (T, error)) (T, error) {
	resultCh := make(chan statusResult, 1)
	go func() {
		value, err := fn()
		resultCh <- statusResult{value: value, err: err}
	}()

	finalModel, err := tea.NewProgram(newStatusModel(title, subtitle, message, resultCh), tea.WithAltScreen()).Run()
	if err != nil {
		var zero T
		return zero, err
	}

	model, ok := finalModel.(statusModel)
	if !ok {
		var zero T
		return zero, nil
	}
	if model.result.err != nil {
		var zero T
		return zero, model.result.err
	}
	if model.result.value == nil {
		var zero T
		return zero, nil
	}
	value, ok := model.result.value.(T)
	if !ok {
		var zero T
		return zero, nil
	}
	return value, nil
}

func RunStatus(title, subtitle, message string, fn func() error) error {
	_, err := RunStatusValue(title, subtitle, message, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}
