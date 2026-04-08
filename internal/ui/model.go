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

type WorktreeItem struct {
	Label      string
	Path       string
	Selectable bool
}

type State struct {
	Screen         Screen
	Worktrees      []WorktreeItem
	Actions        []string
	Branches       []string
	Repositories   []string
	DeleteModes    []string
	Cursor         int
	SelectedPath   string
	SelectedAction string
}

type Model struct {
	State
}

func NewModel(state State) Model {
	model := Model{State: state}
	if model.Screen == ScreenWorktrees {
		model.Cursor = clampWorktreeCursor(model.Cursor, model.Worktrees)
	}
	return model
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
				if m.Screen == ScreenWorktrees {
					m.SelectedAction = "create-workspace-github"
					return m, tea.Quit
				}
			}
		case tea.KeyUp:
			if m.Screen == ScreenWorktrees {
				m.Cursor = moveWorktreeCursor(m.Cursor, m.Worktrees, -1)
			} else if m.Cursor > 0 {
				m.Cursor--
			}
		case tea.KeyDown:
			if m.Screen == ScreenWorktrees {
				m.Cursor = moveWorktreeCursor(m.Cursor, m.Worktrees, 1)
			} else {
				limit := m.listLength()
				if limit > 0 && m.Cursor < limit-1 {
					m.Cursor++
				}
			}
		case tea.KeyEnter:
			switch m.Screen {
			case ScreenWorkspaceGitHubPicker:
				if len(m.Repositories) > 0 {
					m.SelectedPath = m.Repositories[clampIndex(m.Cursor, len(m.Repositories))]
					return m, tea.Quit
				}
				return m, nil
			case ScreenCreateWorktree, ScreenDeleteConfirm:
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
							item := m.Worktrees[clampWorktreeCursor(m.Cursor, m.Worktrees)]
							if item.Selectable {
								m.SelectedPath = item.Path
								return m, tea.Quit
							}
						}
					}
					return m, nil
				}
				if len(m.Worktrees) > 0 {
					item := m.Worktrees[clampWorktreeCursor(m.Cursor, m.Worktrees)]
					if item.Selectable {
						m.SelectedPath = item.Path
						return m, tea.Quit
					}
				}
				return m, nil
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.Screen == ScreenWorktrees {
		if len(m.Worktrees) == 0 {
			return "No worktrees\n\nKeys: c create worktree, q quit"
		}

		var lines []string
		cursor := clampWorktreeCursor(m.Cursor, m.Worktrees)
		for i, item := range m.Worktrees {
			switch {
			case !item.Selectable:
				lines = append(lines, item.Label)
			case i == cursor:
				lines = append(lines, "> "+item.Label)
			default:
				lines = append(lines, "  "+item.Label)
			}
		}
		return strings.Join(lines, "\n") + "\n\nKeys: enter open, c create worktree, q quit"
	}

	items := m.currentItems()
	if len(items) == 0 {
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
		return nil
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

func clampWorktreeCursor(index int, items []WorktreeItem) int {
	if len(items) == 0 {
		return 0
	}
	if index < 0 || index >= len(items) || !items[index].Selectable {
		for i, item := range items {
			if item.Selectable {
				return i
			}
		}
		return 0
	}
	return index
}

func moveWorktreeCursor(current int, items []WorktreeItem, delta int) int {
	if len(items) == 0 {
		return 0
	}
	current = clampWorktreeCursor(current, items)
	for i := current + delta; i >= 0 && i < len(items); i += delta {
		if items[i].Selectable {
			return i
		}
	}
	return current
}
