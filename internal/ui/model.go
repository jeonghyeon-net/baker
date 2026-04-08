package ui

import (
	"fmt"
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
	ScreenOptions               Screen = "options"
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
	Title             string
	Hint              string
	Worktrees         []WorktreeItem
	Actions           []string
	Options           []string
	Branches          []string
	Repositories      []string
	DeleteModes       []string
	Cursor            int
	Width             int
	Height            int
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
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyRunes:
			if m.Screen == ScreenWorktrees {
				switch msg.String() {
				case "a":
					m.SelectedAction = "add-workspace"
					return m, tea.Quit
				case "c":
					if item, ok := m.currentWorktreeItem(); ok && item.WorkspaceName != "" {
						m.SelectedAction = "create-worktree"
						m.SelectedWorkspace = item.WorkspaceName
						return m, tea.Quit
					}
				case "d":
					if item, ok := m.currentWorktreeItem(); ok {
						if item.Selectable {
							m.SelectedAction = "delete-worktree"
							m.SelectedWorkspace = item.WorkspaceName
							m.SelectedPath = item.Path
							m.SelectedBranch = item.BranchName
							return m, tea.Quit
						}
						if item.WorkspaceName != "" {
							m.SelectedAction = "delete-workspace"
							m.SelectedWorkspace = item.WorkspaceName
							return m, tea.Quit
						}
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
			case ScreenOptions:
				if len(m.Options) > 0 {
					m.SelectedAction = m.Options[clampIndex(m.Cursor, len(m.Options))]
					return m, tea.Quit
				}
				return m, nil
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
			return renderScreen("", metaStyle.Render("No workspaces or worktrees"), metaStyle.Render("a add workspace • esc quit"))
		}

		lines := make([]string, 0, len(m.Worktrees))
		cursor := clampIndex(m.Cursor, len(m.Worktrees))
		for i, item := range m.Worktrees {
			style := normalStyle
			if !item.Selectable {
				style = workspaceStyle
			}
			label := style.Render(item.Label)
			if i == cursor {
				label = selectedStyle.Render(item.Label)
			}
			lines = append(lines, label)
		}

		body := renderScrollableLines(lines, cursor, m.bodyHeight("", "enter open • a add workspace • c create worktree • d delete selected • esc quit"))
		return renderScreen("", body, metaStyle.Render("enter open • a add workspace • c create worktree • d delete selected • esc quit"))
	case ScreenOptions:
		return renderScreen(withDefaultTitle(m.Title, "Select option"), renderList(m.Options, m.Cursor, "No options", m.bodyHeight(m.Title, m.Hint)), metaStyle.Render(withDefaultHint(m.Hint, "enter select • esc cancel")))
	case ScreenWorkspaceGitHubPicker:
		return renderScreen(withDefaultTitle(m.Title, "Select repository"), renderList(m.Repositories, m.Cursor, "No repositories", m.bodyHeight(m.Title, m.Hint)), metaStyle.Render(withDefaultHint(m.Hint, "enter select • esc cancel")))
	case ScreenCreateWorktree:
		return renderScreen(withDefaultTitle(m.Title, "Select branch"), renderList(m.Branches, m.Cursor, "No branches", m.bodyHeight(m.Title, m.Hint)), metaStyle.Render(withDefaultHint(m.Hint, "enter select • esc cancel")))
	case ScreenDeleteConfirm:
		return renderScreen(withDefaultTitle(m.Title, "Delete worktree"), renderList(m.DeleteModes, m.Cursor, "No delete modes", m.bodyHeight(m.Title, m.Hint)), metaStyle.Render(withDefaultHint(m.Hint, "enter select • esc cancel")))
	default:
		return ""
	}
}

func (m Model) listLength() int {
	switch m.Screen {
	case ScreenWorktrees:
		return len(m.Worktrees)
	case ScreenOptions:
		return len(m.Options)
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

func (m Model) bodyHeight(title, hint string) int {
	if m.Height <= 0 {
		return 0
	}

	reserved := 0
	if title != "" {
		reserved += 2
	}
	if hint != "" {
		reserved += 2
	}
	if m.Height-reserved < 1 {
		return 1
	}
	return m.Height - reserved
}

func renderScreen(title, body, hint string) string {
	parts := make([]string, 0, 3)
	if title != "" {
		parts = append(parts, titleStyle.Render(title))
	}
	if body != "" {
		parts = append(parts, body)
	}
	if hint != "" {
		parts = append(parts, hint)
	}
	return strings.Join(parts, "\n\n")
}

func withDefaultTitle(title, fallback string) string {
	if title != "" {
		return title
	}
	return fallback
}

func withDefaultHint(hint, fallback string) string {
	if hint != "" {
		return hint
	}
	return fallback
}

func renderList(items []string, cursor int, empty string, maxBodyLines int) string {
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
	return renderScrollableLines(lines, cursor, maxBodyLines)
}

func renderScrollableLines(lines []string, cursor, maxBodyLines int) string {
	if len(lines) == 0 {
		return ""
	}
	if maxBodyLines <= 0 || len(lines) <= maxBodyLines {
		return strings.Join(lines, "\n")
	}

	listLines := maxBodyLines - 1
	if listLines < 1 {
		listLines = 1
	}
	start, end := visibleRange(len(lines), cursor, listLines)
	body := strings.Join(lines[start:end], "\n")
	status := metaStyle.Render(fmt.Sprintf("%d-%d/%d", start+1, end, len(lines)))
	return body + "\n" + status
}

func visibleRange(length, cursor, size int) (int, int) {
	if length <= 0 {
		return 0, 0
	}
	if size <= 0 || size >= length {
		return 0, length
	}

	cursor = clampIndex(cursor, length)
	start := cursor - size/2
	if start < 0 {
		start = 0
	}
	end := start + size
	if end > length {
		end = length
		start = end - size
	}
	if start < 0 {
		start = 0
	}
	return start, end
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
