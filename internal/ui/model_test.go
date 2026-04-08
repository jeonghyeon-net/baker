package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCreateShortcutSelectsCreateActionFromWorktreeScreen(t *testing.T) {
	model := NewModel(State{Screen: ScreenWorktrees})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	updated := next.(Model)

	if updated.SelectedAction != "create-workspace-github" {
		t.Fatalf("SelectedAction = %q", updated.SelectedAction)
	}
}

func TestEnterSelectsHighlightedWorktree(t *testing.T) {
	model := NewModel(State{
		Screen: ScreenWorktrees,
		Worktrees: []WorktreeItem{
			{Label: "baker", Selectable: false},
			{Label: "  main", Path: "/tmp/baker/main", Selectable: true},
			{Label: "  feature-login", Path: "/tmp/baker/feature-login", Selectable: true},
		},
		Cursor: 2,
	})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := next.(Model)

	if updated.SelectedPath != "/tmp/baker/feature-login" {
		t.Fatalf("SelectedPath = %q", updated.SelectedPath)
	}
}

func TestWorktreeCursorSkipsWorkspaceHeaders(t *testing.T) {
	model := NewModel(State{
		Screen: ScreenWorktrees,
		Worktrees: []WorktreeItem{
			{Label: "baker", Selectable: false},
			{Label: "  main", Path: "/tmp/baker/main", Selectable: true},
			{Label: "other", Selectable: false},
			{Label: "  fix", Path: "/tmp/other/fix", Selectable: true},
		},
		Cursor: 1,
	})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := next.(Model)
	if updated.Cursor != 3 {
		t.Fatalf("Cursor = %d", updated.Cursor)
	}
}

func TestWorktreeViewShowsTreeLabelsNotFullPaths(t *testing.T) {
	model := NewModel(State{
		Screen: ScreenWorktrees,
		Worktrees: []WorktreeItem{
			{Label: "baker", Selectable: false},
			{Label: "  main", Path: "/Users/me/.pi/worktrees/baker/main", Selectable: true},
		},
	})

	view := model.View()
	if !strings.Contains(view, "baker") || !strings.Contains(view, "  main") {
		t.Fatalf("view = %q", view)
	}
	if strings.Contains(view, "/Users/me/.pi/worktrees") {
		t.Fatalf("view leaked full path: %q", view)
	}
}
