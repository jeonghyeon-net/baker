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

const NewBranchOption = "+ 새 브랜치 만들기"

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

const (
	defaultPanelWidth = 112
	minPanelWidth     = 72
)

var (
	appStyle = lipgloss.NewStyle().
			Padding(1, 2)
	panelStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2)
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255"))
	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("109"))
	workspaceStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("117"))
	worktreeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
	selectedTextStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("60")).
				Padding(0, 1)
	selectedMetaStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("153"))
	indicatorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("111"))
	metaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
	pillStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("238")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)
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
			switch msg.String() {
			case "j":
				return m.moveCursor(1), nil
			case "k":
				return m.moveCursor(-1), nil
			}
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
			return m.moveCursor(-1), nil
		case tea.KeyDown:
			return m.moveCursor(1), nil
		case tea.KeyPgUp:
			return m.moveCursor(-m.pageStep()), nil
		case tea.KeyPgDown:
			return m.moveCursor(m.pageStep()), nil
		case tea.KeyLeft:
			if m.Screen == ScreenWorktrees {
				return m.jumpWorkspace(-1), nil
			}
			return m, nil
		case tea.KeyRight:
			if m.Screen == ScreenWorktrees {
				return m.jumpWorkspace(1), nil
			}
			return m, nil
		case tea.KeyHome:
			m.Cursor = 0
			return m, nil
		case tea.KeyEnd:
			if length := m.listLength(); length > 0 {
				m.Cursor = length - 1
			}
			return m, nil
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
	title, subtitle, footer := m.screenChrome()
	body := m.screenBody(title, footer)

	content := renderFrame(renderPanel(title, subtitle, body, footer), m.Width)
	return renderRoot(content, m.Width, m.Height)
}

func (m Model) screenChrome() (string, string, string) {
	switch m.Screen {
	case ScreenWorktrees:
		return "워크트리 목록", "워크스페이스별 트리", m.worktreeScreenHint()
	case ScreenOptions:
		return withDefaultTitle(m.Title, "항목 선택"), "작업 선택", withDefaultHint(m.Hint, "↑↓/j/k 이동 • enter 선택 • esc 취소")
	case ScreenWorkspaceGitHubPicker:
		return withDefaultTitle(m.Title, "저장소 선택"), "GitHub 저장소 선택", withDefaultHint(m.Hint, "↑↓/j/k 이동 • enter 선택 • esc 취소")
	case ScreenCreateWorktree:
		return withDefaultTitle(m.Title, "브랜치 선택"), "브랜치 선택", withDefaultHint(m.Hint, "↑↓/j/k 이동 • enter 선택 • esc 취소")
	case ScreenDeleteConfirm:
		return withDefaultTitle(m.Title, "워크트리 삭제"), "삭제 방식 선택", withDefaultHint(m.Hint, "↑↓/j/k 이동 • enter 선택 • esc 취소")
	default:
		return "", "", ""
	}
}

func (m Model) screenBody(title, footer string) string {
	bodyHeight := m.bodyHeight(title, footer)

	switch m.Screen {
	case ScreenWorktrees:
		if len(m.Worktrees) == 0 {
			return metaStyle.Render("아직 워크스페이스가 없습니다. a 키로 추가하세요.")
		}
		lines := make([]string, 0, len(m.Worktrees))
		cursor := clampIndex(m.Cursor, len(m.Worktrees))
		for i, item := range m.Worktrees {
			lines = append(lines, renderTreeLine(item, i == cursor))
		}
		return renderScrollableLines(lines, cursor, bodyHeight)
	case ScreenOptions:
		return renderList(m.Options, m.Cursor, "표시할 항목이 없습니다.", bodyHeight)
	case ScreenWorkspaceGitHubPicker:
		return renderList(m.Repositories, m.Cursor, "표시할 저장소가 없습니다.", bodyHeight)
	case ScreenCreateWorktree:
		return renderList(m.Branches, m.Cursor, "표시할 브랜치가 없습니다.", bodyHeight)
	case ScreenDeleteConfirm:
		return renderList(m.DeleteModes, m.Cursor, "표시할 삭제 방식이 없습니다.", bodyHeight)
	default:
		return ""
	}
}

func renderTreeLine(item WorktreeItem, selected bool) string {
	indicator := metaStyle.Render("  ")
	textStyle := worktreeStyle

	if !item.Selectable {
		textStyle = workspaceStyle
	}

	if selected {
		indicator = indicatorStyle.Render("› ")
		return indicator + selectedTextStyle.Render(item.Label)
	}

	return indicator + textStyle.Render(item.Label)
}

func renderList(items []string, cursor int, empty string, maxBodyLines int) string {
	if len(items) == 0 {
		return metaStyle.Render(empty)
	}
	cursor = clampIndex(cursor, len(items))
	lines := make([]string, 0, len(items))
	for i, item := range items {
		lines = append(lines, renderListLine(item, i == cursor))
	}
	return renderScrollableLines(lines, cursor, maxBodyLines)
}

func renderListLine(item string, selected bool) string {
	if selected {
		return indicatorStyle.Render("› ") + selectedTextStyle.Render(item)
	}
	return metaStyle.Render("  ") + worktreeStyle.Render(item)
}

func renderPanel(title, subtitle, body, footer string) string {
	sections := make([]string, 0, 4)
	if title != "" {
		header := titleStyle.Render(title)
		if subtitle != "" {
			header = lipgloss.JoinHorizontal(lipgloss.Center, header, "  ", subtitleStyle.Render(subtitle))
		}
		sections = append(sections, header)
	}
	if body != "" {
		sections = append(sections, body)
	}
	if footer != "" {
		sections = append(sections, footer)
	}
	return strings.Join(sections, "\n\n")
}

func (m Model) worktreeScreenHint() string {
	item, ok := m.currentWorktreeItem()
	if !ok {
		return renderActionPanel([]string{"a  워크스페이스 추가", "← →  워크스페이스 이동", "esc  종료"})
	}
	if item.Selectable {
		return renderActionPanel([]string{
			"enter  현재 워크트리 열기",
			"← →  워크스페이스 이동",
			fmt.Sprintf("c  %s에 새 워크트리 만들기", item.WorkspaceName),
			"d  현재 워크트리 삭제",
			"esc  종료",
		})
	}
	if item.WorkspaceName != "" {
		return renderActionPanel([]string{
			"a  새 워크스페이스 추가",
			"← →  워크스페이스 이동",
			fmt.Sprintf("c  %s에 새 워크트리 만들기", item.WorkspaceName),
			fmt.Sprintf("d  %s 삭제", item.WorkspaceName),
			"esc  종료",
		})
	}
	return renderActionPanel([]string{"a  워크스페이스 추가", "← →  워크스페이스 이동", "esc  종료"})
}

func renderActionPanel(lines []string) string {
	rendered := make([]string, 0, len(lines))
	for _, line := range lines {
		rendered = append(rendered, metaStyle.Render(line))
	}
	return strings.Join(rendered, "\n")
}

func shouldShowBranchDetail(worktreeName, branchName string) bool {
	if worktreeName == "" || branchName == "" {
		return false
	}
	if worktreeName == branchName {
		return false
	}
	normalizedBranch := strings.NewReplacer("/", "-", `\\`, "-").Replace(branchName)
	return normalizedBranch != worktreeName
}

func (m Model) jumpWorkspace(delta int) Model {
	if m.Screen != ScreenWorktrees || len(m.Worktrees) == 0 || delta == 0 {
		return m
	}

	workspaceIndexes := make([]int, 0)
	for i, item := range m.Worktrees {
		if !item.Selectable && item.WorkspaceName != "" {
			workspaceIndexes = append(workspaceIndexes, i)
		}
	}
	if len(workspaceIndexes) == 0 {
		return m
	}

	currentGroup := 0
	for i, index := range workspaceIndexes {
		if index <= m.Cursor {
			currentGroup = i
		} else {
			break
		}
	}

	nextGroup := wrapIndex(currentGroup+delta, len(workspaceIndexes))
	m.Cursor = workspaceIndexes[nextGroup]
	return m
}

func (m Model) moveCursor(delta int) Model {
	if delta == 0 {
		return m
	}
	length := m.listLength()
	if length == 0 {
		return m
	}
	m.Cursor = wrapIndex(m.Cursor+delta, length)
	return m
}

func (m Model) pageStep() int {
	bodyHeight := m.bodyHeight("", "") - 1
	if bodyHeight < 5 {
		return 5
	}
	return bodyHeight
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

func (m Model) bodyHeight(title, footer string) int {
	if m.Height <= 0 {
		return 0
	}
	reserved := 8
	if title != "" {
		reserved += 2
	}
	if footer != "" {
		reserved += 2
	}
	available := m.Height - reserved
	if available < 3 {
		return 3
	}
	return available
}

func renderRoot(content string, windowWidth, windowHeight int) string {
	content = appStyle.Render(content)
	if windowWidth > 0 && windowHeight > 0 {
		return lipgloss.Place(windowWidth, windowHeight, lipgloss.Center, lipgloss.Center, content)
	}
	if windowWidth > 0 {
		return lipgloss.PlaceHorizontal(windowWidth, lipgloss.Center, content)
	}
	return content
}

func renderFrame(content string, windowWidth int) string {
	return panelStyle.Width(panelWidth(windowWidth)).Render(content)
}

func panelWidth(windowWidth int) int {
	if windowWidth <= 0 {
		return defaultPanelWidth
	}

	available := windowWidth - 6
	if available < minPanelWidth {
		if available < 32 {
			return 32
		}
		return available
	}
	if available > defaultPanelWidth {
		return defaultPanelWidth
	}
	return available
}

func panelInnerWidth(windowWidth int) int {
	width := panelWidth(windowWidth) - 6
	if width < 20 {
		return 20
	}
	return width
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

func wrapIndex(index int, length int) int {
	if length <= 0 {
		return 0
	}
	index %= length
	if index < 0 {
		index += length
	}
	return index
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
