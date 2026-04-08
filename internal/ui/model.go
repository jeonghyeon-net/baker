package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Screen string

const (
	ScreenWorktrees             Screen = "worktrees"
	ScreenCreateWorktree        Screen = "create-worktree"
	ScreenWorkspaceGitHubPicker Screen = "workspace-github-picker"
	ScreenDeleteConfirm         Screen = "delete-confirm"
)

const NewBranchOption = "+ create new branch"

type WorktreeItem struct {
	Label         string
	Path          string
	WorkspaceName string
	WorktreeName  string
	BranchName    string
	Selectable    bool
}

type State struct {
	Screen            Screen
	Worktrees         []WorktreeItem
	Actions           []string
	Branches          []string
	Repositories      []string
	DeleteModes       []string
	Cursor            int
	SelectedPath      string
	SelectedAction    string
	SelectedWorkspace string
	SelectedBranch    string
}

type Model struct {
	State
}

var (
	workspaceStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	selectedStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62")).Padding(0, 1)
	normalStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	metaStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
)

func NewModel(state State) Model {
	model := Model{State: state}
	model.Cursor = clampIndex(model.Cursor, model.listLength())
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
			case "a":
				if m.Screen == ScreenWorktrees {
					m.SelectedAction = "add-workspace"
					return m, tea.Quit
				}
			case "c":
				if m.Screen == ScreenWorktrees {
					if item, ok := m.currentWorktreeItem(); ok && item.WorkspaceName != "" {
						m.SelectedAction = "create-worktree"
						m.SelectedWorkspace = item.WorkspaceName
						return m, tea.Quit
					}
				}
			case "d":
				if m.Screen == ScreenWorktrees {
					if item, ok := m.currentWorktreeItem(); ok && item.Selectable {
						m.SelectedAction = "delete-worktree"
						m.SelectedWorkspace = item.WorkspaceName
						m.SelectedPath = item.Path
						m.SelectedBranch = item.BranchName
						return m, tea.Quit
					}
				}
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
			case ScreenWorkspaceGitHubPicker:
				if len(m.Repositories) > 0 {
					m.SelectedPath = m.Repositories[clampIndex(m.Cursor, len(m.Repositories))]
					return m, tea.Quit
				}
				return m, nil
			case ScreenCreateWorktree:
				if len(m.Branches) > 0 {
					selection := m.Branches[clampIndex(m.Cursor, len(m.Branches))]
					if selection == NewBranchOption {
						m.SelectedAction = "create-new-branch"
					} else {
						m.SelectedBranch = selection
					}
					return m, tea.Quit
				}
				return m, nil
			case ScreenDeleteConfirm:
				if len(m.DeleteModes) > 0 {
					m.SelectedAction = m.DeleteModes[clampIndex(m.Cursor, len(m.DeleteModes))]
					return m, tea.Quit
				}
				return m, nil
			default:
				if item, ok := m.currentWorktreeItem(); ok && item.Selectable {
					m.SelectedPath = item.Path
					return m, tea.Quit
				}
				return m, nil
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	switch m.Screen {
	case ScreenWorktrees:
		if len(m.Worktrees) == 0 {
			return titleStyle.Render("baker") + "\n\n" + metaStyle.Render("No workspaces or worktrees") + "\n\n" + metaStyle.Render("Keys: a add workspace, q quit")
		}
		var lines []string
		cursor := clampIndex(m.Cursor, len(m.Worktrees))
		for i, item := range m.Worktrees {
			if item.Selectable {
				label := normalStyle.Render(item.Label)
				if i == cursor {
					label = selectedStyle.Render(item.Label)
				}
				lines = append(lines, label)
				continue
			}
			label := workspaceStyle.Render(item.Label)
			if i == cursor {
				label = selectedStyle.Render(item.Label)
			}
			lines = append(lines, label)
		}
		return titleStyle.Render("baker") + "\n\n" + strings.Join(lines, "\n") + "\n\n" + metaStyle.Render("Keys: enter open, a add workspace, c create worktree, d delete worktree, q quit")
	case ScreenWorkspaceGitHubPicker:
		return titleStyle.Render("select repository") + "\n\n" + renderList(m.Repositories, m.Cursor, "No repositories")
	case ScreenCreateWorktree:
		return titleStyle.Render("select branch") + "\n\n" + renderList(m.Branches, m.Cursor, "No branches")
	case ScreenDeleteConfirm:
		return titleStyle.Render("delete mode") + "\n\n" + renderList(m.DeleteModes, m.Cursor, "No delete modes")
	default:
		return ""
	}
}

func (m Model) listLength() int {
	switch m.Screen {
	case ScreenWorktrees:
		return len(m.Worktrees)
	case ScreenWorkspaceGitHubPicker:
		return len(m.Repositories)
	case ScreenCreateWorktree:
		return len(m.Branches)
	case ScreenDeleteConfirm:
		return len(m.DeleteModes)
	default:
		return 0
	}
}

func (m Model) currentWorktreeItem() (WorktreeItem, bool) {
	if m.Screen != ScreenWorktrees || len(m.Worktrees) == 0 {
		return WorktreeItem{}, false
	}
	return m.Worktrees[clampIndex(m.Cursor, len(m.Worktrees))], true
}

func renderList(items []string, cursor int, empty string) string {
	if len(items) == 0 {
		return metaStyle.Render(empty)
	}
	cursor = clampIndex(cursor, len(items))
	lines := make([]string, 0, len(items))
	for i, item := range items {
		if i == cursor {
			lines = append(lines, selectedStyle.Render(item))
			continue
		}
		lines = append(lines, normalStyle.Render(item))
	}
	return strings.Join(lines, "\n")
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
