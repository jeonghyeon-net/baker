package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestEnterSelectsHighlightedMainMenuAction(t *testing.T) {
	model := NewModel(State{Screen: ScreenMainMenu, Actions: []string{"open", "create-workspace-github", "quit"}, Cursor: 1})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := next.(Model)

	if updated.SelectedAction != "create-workspace-github" {
		t.Fatalf("SelectedAction = %q", updated.SelectedAction)
	}
}

func TestEnterSelectsHighlightedWorktree(t *testing.T) {
	model := NewModel(State{
		Screen:    ScreenWorktrees,
		Worktrees: []string{"main", "feature-login"},
		Cursor:    1,
	})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := next.(Model)

	if updated.SelectedPath != "feature-login" {
		t.Fatalf("SelectedPath = %q", updated.SelectedPath)
	}
}

func TestWorkspaceGitHubPickerActionEntersRepoList(t *testing.T) {
	model := NewModel(State{Screen: ScreenWorktrees, Actions: []string{"open", "create-worktree", "create-workspace-github", "delete-worktree"}, Cursor: 2})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := next.(Model)

	if updated.Screen != ScreenWorkspaceGitHubPicker {
		t.Fatalf("Screen = %q", updated.Screen)
	}
}

func TestDeleteActionEntersConfirmationScreenWithModes(t *testing.T) {
	model := NewModel(State{Screen: ScreenWorktrees, Actions: []string{"open", "create-worktree", "create-workspace-github", "delete-worktree"}, Cursor: 3})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := next.(Model)

	if updated.Screen != ScreenDeleteConfirm {
		t.Fatalf("Screen = %q", updated.Screen)
	}
	if len(updated.DeleteModes) != 3 {
		t.Fatalf("DeleteModes = %#v", updated.DeleteModes)
	}
}
