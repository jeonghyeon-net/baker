package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Screen string

const (
	ScreenWorktrees             Screen = "worktrees"
	ScreenCreateWorktree        Screen = "create-worktree"
	ScreenWorkspaceGitHubPicker Screen = "workspace-github-picker"
	ScreenDeleteConfirm         Screen = "delete-confirm"
)

type State struct {
	Screen       Screen
	Worktrees    []string
	Actions      []string
	Branches     []string
	Repositories []string
	DeleteModes  []string
	Cursor       int
	SelectedPath string
}

type Model struct {
	State
}

func NewModel(state State) Model {
	return Model{State: state}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyRunes:
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "c":
				m.Screen = ScreenCreateWorktree
				m.Cursor = 0
				return m, nil
			case "g":
				m.Screen = ScreenWorkspaceGitHubPicker
				m.Cursor = 0
				return m, nil
			case "d":
				m.Screen = ScreenDeleteConfirm
				m.DeleteModes = []string{"worktree-only", "worktree-and-local-branch", "worktree-local-and-remote-branch"}
				m.Cursor = 0
				return m, nil
			}
		case tea.KeyUp:
			if m.Cursor > 0 {
				m.Cursor--
			}
		case tea.KeyDown:
			limit := m.listLength()
			if limit > 0 && m.Cursor < limit-1 {
				m.Cursor++
			}
		case tea.KeyEnter:
			switch m.Screen {
			case ScreenDeleteConfirm, ScreenWorkspaceGitHubPicker, ScreenCreateWorktree:
				return m, nil
			default:
				if len(m.Actions) > 0 {
					index := clampIndex(m.Cursor, len(m.Actions))
					switch m.Actions[index] {
					case "create-worktree":
						m.Screen = ScreenCreateWorktree
						m.Cursor = 0
					case "create-workspace-github":
						m.Screen = ScreenWorkspaceGitHubPicker
						m.Cursor = 0
					case "delete-worktree":
						m.Screen = ScreenDeleteConfirm
						m.DeleteModes = []string{"worktree-only", "worktree-and-local-branch", "worktree-local-and-remote-branch"}
						m.Cursor = 0
					case "open":
						if len(m.Worktrees) > 0 {
							m.SelectedPath = m.Worktrees[clampIndex(m.Cursor, len(m.Worktrees))]
							return m, tea.Quit
						}
					}
					return m, nil
				}
				if len(m.Worktrees) > 0 {
					m.SelectedPath = m.Worktrees[clampIndex(m.Cursor, len(m.Worktrees))]
					return m, tea.Quit
				}
				return m, nil
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	items := m.currentItems()
	if len(items) == 0 {
		if m.Screen == ScreenWorktrees {
			return "No worktrees\n\nKeys: enter open, g GitHub workspace picker, c create worktree, d delete flow, q quit"
		}
		return "No items"
	}

	var lines []string
	for i, item := range items {
		prefix := "  "
		if i == m.Cursor {
			prefix = "> "
		}
		lines = append(lines, prefix+item)
	}
	return strings.Join(lines, "\n")
}

func (m Model) currentItems() []string {
	switch m.Screen {
	case ScreenCreateWorktree:
		return m.Branches
	case ScreenWorkspaceGitHubPicker:
		return m.Repositories
	case ScreenDeleteConfirm:
		return m.DeleteModes
	default:
		if len(m.Actions) > 0 {
			return m.Actions
		}
		return m.Worktrees
	}
}

func (m Model) listLength() int {
	return len(m.currentItems())
}

func clampIndex(index int, length int) int {
	if length <= 0 {
		return 0
	}
	if index < 0 || index >= length {
		return 0
	}
	return index
}
