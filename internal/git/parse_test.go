package git

import (
	"testing"
)

func TestParseBranches(t *testing.T) {
	output := "main\tlocal\torigin\nfeature/login\tremote\torigin\n"

	branches, err := ParseBranches(output)
	if err != nil {
		t.Fatalf("ParseBranches returned error: %v", err)
	}

	if len(branches) != 2 {
		t.Fatalf("expected 2 branches, got %d", len(branches))
	}

	second := branches[1]
	if second.Name != "feature/login" {
		t.Fatalf("expected second branch name %q, got %q", "feature/login", second.Name)
	}
	if second.Source != "remote" {
		t.Fatalf("expected second branch source %q, got %q", "remote", second.Source)
	}

}

func TestParseWorktrees(t *testing.T) {
	t.Run("branch worktree", func(t *testing.T) {
		output := "worktree /tmp/wt/main\nHEAD 0123456\nbranch refs/heads/main\n\n"

		worktrees, err := ParseWorktrees(output)
		if err != nil {
			t.Fatalf("ParseWorktrees returned error: %v", err)
		}

		if len(worktrees) != 1 {
			t.Fatalf("expected 1 worktree, got %d", len(worktrees))
		}

		if worktrees[0].BranchName != "main" {
			t.Fatalf("expected branch name %q, got %q", "main", worktrees[0].BranchName)
		}
	})

	t.Run("bare worktree does not error", func(t *testing.T) {
		output := "worktree /tmp/wt/bare\nbare\n\n"

		worktrees, err := ParseWorktrees(output)
		if err != nil {
			t.Fatalf("ParseWorktrees returned error for bare block: %v", err)
		}

		if len(worktrees) != 0 {
			t.Fatalf("expected bare worktree block to be skipped, got %d worktrees", len(worktrees))
		}
	})
}
