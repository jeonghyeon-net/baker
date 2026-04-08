package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAddShortcutSelectsAddWorkspaceAction(t *testing.T) {
	model := NewModel(State{Screen: ScreenWorktrees})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	updated := next.(Model)

	if updated.SelectedAction != "add-workspace" {
		t.Fatalf("SelectedAction = %q", updated.SelectedAction)
	}
}

func TestCreateShortcutSelectsCreateActionFromSelectedWorkspace(t *testing.T) {
	model := NewModel(State{
		Screen: ScreenWorktrees,
		Worktrees: []WorktreeItem{
			{Label: "baker", WorkspaceName: "baker"},
			{Label: "  main", WorkspaceName: "baker", Path: "/tmp/baker/main", BranchName: "main", Selectable: true},
		},
		Cursor: 0,
	})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	updated := next.(Model)

	if updated.SelectedAction != "create-worktree" {
		t.Fatalf("SelectedAction = %q", updated.SelectedAction)
	}
	if updated.SelectedWorkspace != "baker" {
		t.Fatalf("SelectedWorkspace = %q", updated.SelectedWorkspace)
	}
}

func TestDeleteShortcutSelectsDeleteActionFromWorktree(t *testing.T) {
	model := NewModel(State{
		Screen: ScreenWorktrees,
		Worktrees: []WorktreeItem{
			{Label: "baker", WorkspaceName: "baker"},
			{Label: "  feature-login", WorkspaceName: "baker", Path: "/tmp/baker/feature-login", BranchName: "feature/login", Selectable: true},
		},
		Cursor: 1,
	})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	updated := next.(Model)

	if updated.SelectedAction != "delete-worktree" {
		t.Fatalf("SelectedAction = %q", updated.SelectedAction)
	}
	if updated.SelectedPath != "/tmp/baker/feature-login" {
		t.Fatalf("SelectedPath = %q", updated.SelectedPath)
	}
	if updated.SelectedBranch != "feature/login" {
		t.Fatalf("SelectedBranch = %q", updated.SelectedBranch)
	}
}

func TestDeleteShortcutSelectsDeleteWorkspaceActionFromWorkspaceHeader(t *testing.T) {
	model := NewModel(State{
		Screen: ScreenWorktrees,
		Worktrees: []WorktreeItem{
			{Label: "baker", WorkspaceName: "baker"},
			{Label: "  main", WorkspaceName: "baker", Path: "/tmp/baker/main", BranchName: "main", Selectable: true},
		},
		Cursor: 0,
	})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	updated := next.(Model)

	if updated.SelectedAction != "delete-workspace" {
		t.Fatalf("SelectedAction = %q", updated.SelectedAction)
	}
	if updated.SelectedWorkspace != "baker" {
		t.Fatalf("SelectedWorkspace = %q", updated.SelectedWorkspace)
	}
}

func TestEnterSelectsHighlightedWorktree(t *testing.T) {
	model := NewModel(State{
		Screen: ScreenWorktrees,
		Worktrees: []WorktreeItem{
			{Label: "baker", WorkspaceName: "baker"},
			{Label: "  main", WorkspaceName: "baker", Path: "/tmp/baker/main", BranchName: "main", Selectable: true},
			{Label: "  feature-login", WorkspaceName: "baker", Path: "/tmp/baker/feature-login", BranchName: "feature/login", Selectable: true},
		},
		Cursor: 2,
	})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := next.(Model)

	if updated.SelectedPath != "/tmp/baker/feature-login" {
		t.Fatalf("SelectedPath = %q", updated.SelectedPath)
	}
}

func TestWorktreeViewShowsTreeLabelsNotFullPaths(t *testing.T) {
	model := NewModel(State{
		Screen: ScreenWorktrees,
		Worktrees: []WorktreeItem{
			{Label: "baker", WorkspaceName: "baker"},
			{Label: "  main", WorkspaceName: "baker", Path: "/Users/me/.pi/worktrees/baker/main", BranchName: "main", Selectable: true},
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

func TestEnterSelectsBranchInCreateScreen(t *testing.T) {
	model := NewModel(State{Screen: ScreenCreateWorktree, Branches: []string{"main", "feature/login"}, Cursor: 1})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := next.(Model)

	if updated.SelectedBranch != "feature/login" {
		t.Fatalf("SelectedBranch = %q", updated.SelectedBranch)
	}
}

func TestEnterSelectsNewBranchActionInCreateScreen(t *testing.T) {
	model := NewModel(State{Screen: ScreenCreateWorktree, Branches: []string{NewBranchOption, "main"}})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := next.(Model)

	if updated.SelectedAction != "create-new-branch" {
		t.Fatalf("SelectedAction = %q", updated.SelectedAction)
	}
}

func TestEnterSelectsDeleteMode(t *testing.T) {
	model := NewModel(State{Screen: ScreenDeleteConfirm, DeleteModes: []string{"worktree-and-local-branch", "worktree-local-and-remote-branch"}, Cursor: 1})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := next.(Model)

	if updated.SelectedAction != "worktree-local-and-remote-branch" {
		t.Fatalf("SelectedAction = %q", updated.SelectedAction)
	}
}

func TestEscQuitsAcrossScreens(t *testing.T) {
	model := NewModel(State{Screen: ScreenOptions, Options: []string{"one"}})

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := next.(Model)

	if updated.SelectedAction != "" {
		t.Fatalf("SelectedAction = %q", updated.SelectedAction)
	}
	if cmd == nil {
		t.Fatal("expected quit command on esc")
	}
}

func TestListViewShowsWindowStatusWhenItemsOverflow(t *testing.T) {
	model := NewModel(State{
		Screen:       ScreenWorkspaceGitHubPicker,
		Title:        "Select repository",
		Hint:         "enter select • esc cancel",
		Height:       8,
		Cursor:       8,
		Repositories: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
	})

	view := model.View()
	if !strings.Contains(view, "8") || !strings.Contains(view, "9") {
		t.Fatalf("view = %q", view)
	}
	if !strings.Contains(view, "8-10/10") {
		t.Fatalf("view missing window status: %q", view)
	}
}
