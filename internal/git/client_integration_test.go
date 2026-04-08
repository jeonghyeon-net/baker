package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jeonghyeon-net/baker/internal/domain"
)

func TestClientIntegrationCloneBareFetchAllListBranches(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	remotePath := filepath.Join(tempDir, "remote.git")
	runGit(t, tempDir, "init", "--bare", remotePath)

	seedPath := filepath.Join(tempDir, "seed")
	runGit(t, tempDir, "clone", remotePath, seedPath)
	runGit(t, seedPath, "config", "user.name", "Baker Test")
	runGit(t, seedPath, "config", "user.email", "baker-test@example.com")
	runGit(t, seedPath, "checkout", "-b", "main")

	readmePath := filepath.Join(seedPath, "README.md")
	if err := os.WriteFile(readmePath, []byte("hello from baker\n"), 0o644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}

	runGit(t, seedPath, "add", "README.md")
	runGit(t, seedPath, "commit", "-m", "seed remote")
	runGit(t, seedPath, "push", "--set-upstream", "origin", "main")
	runGit(t, remotePath, "symbolic-ref", "HEAD", "refs/heads/main")

	localBarePath := filepath.Join(tempDir, "local.git")
	client := Client{Runner: defaultRunner{}}

	if err := client.CloneBare(ctx, remotePath, localBarePath); err != nil {
		t.Fatalf("CloneBare returned error: %v", err)
	}

	if err := client.FetchAll(ctx, localBarePath); err != nil {
		t.Fatalf("FetchAll returned error: %v", err)
	}

	branches, err := client.ListBranches(ctx, localBarePath)
	if err != nil {
		t.Fatalf("ListBranches returned error: %v", err)
	}

	assertBranchNames(t, branches, "main")

	seedFeaturePath := filepath.Join(tempDir, "seed-feature")
	runGit(t, tempDir, "clone", remotePath, seedFeaturePath)
	runGit(t, seedFeaturePath, "config", "user.name", "Baker Test")
	runGit(t, seedFeaturePath, "config", "user.email", "baker-test@example.com")
	runGit(t, seedFeaturePath, "checkout", "-b", "feature/refresh", "origin/main")
	runGit(t, seedFeaturePath, "push", "--set-upstream", "origin", "feature/refresh")

	if err := client.FetchAll(ctx, localBarePath); err != nil {
		t.Fatalf("FetchAll after remote branch creation returned error: %v", err)
	}

	branches, err = client.ListBranches(ctx, localBarePath)
	if err != nil {
		t.Fatalf("ListBranches after remote branch creation returned error: %v", err)
	}

	assertBranchNames(t, branches, "feature/refresh", "main")
}

func TestClientIntegrationAddNewBranchWorktreeAndPushBranch(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	remotePath := filepath.Join(tempDir, "remote.git")
	runGit(t, tempDir, "init", "--bare", remotePath)

	seedPath := filepath.Join(tempDir, "seed")
	runGit(t, tempDir, "clone", remotePath, seedPath)
	runGit(t, seedPath, "config", "user.name", "Baker Test")
	runGit(t, seedPath, "config", "user.email", "baker-test@example.com")
	runGit(t, seedPath, "checkout", "-b", "main")

	readmePath := filepath.Join(seedPath, "README.md")
	if err := os.WriteFile(readmePath, []byte("hello from baker\n"), 0o644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}

	runGit(t, seedPath, "add", "README.md")
	runGit(t, seedPath, "commit", "-m", "seed remote")
	runGit(t, seedPath, "push", "--set-upstream", "origin", "main")
	runGit(t, remotePath, "symbolic-ref", "HEAD", "refs/heads/main")

	localBarePath := filepath.Join(tempDir, "local.git")
	worktreePath := filepath.Join(tempDir, "worktrees", "feature-login")
	client := Client{Runner: defaultRunner{}}

	if err := client.CloneBare(ctx, remotePath, localBarePath); err != nil {
		t.Fatalf("CloneBare returned error: %v", err)
	}

	if err := client.FetchAll(ctx, localBarePath); err != nil {
		t.Fatalf("FetchAll returned error: %v", err)
	}

	if err := client.AddNewBranchWorktree(ctx, localBarePath, "main", "feature/login", worktreePath); err != nil {
		t.Fatalf("AddNewBranchWorktree returned error: %v", err)
	}

	if err := client.PushBranch(ctx, worktreePath, "feature/login"); err != nil {
		t.Fatalf("PushBranch returned error: %v", err)
	}

	runGit(t, remotePath, "show-ref", "--verify", "refs/heads/feature/login")

	if err := client.DeleteRemoteBranch(ctx, localBarePath, "feature/login"); err != nil {
		t.Fatalf("DeleteRemoteBranch returned error: %v", err)
	}

	if err := runGitExpectError(remotePath, "show-ref", "--verify", "refs/heads/feature/login"); err == nil {
		t.Fatal("expected feature/login to be deleted from remote")
	}
}

func assertBranchNames(t *testing.T, branches []domain.BranchRef, expectedNames ...string) {
	t.Helper()

	if len(branches) != len(expectedNames) {
		t.Fatalf("expected %d branches, got %d: %#v", len(expectedNames), len(branches), branches)
	}

	for i, expected := range expectedNames {
		branch := branches[i]
		if branch.Name != expected {
			t.Fatalf("expected branch %d name %q, got %q", i, expected, branch.Name)
		}
		if branch.Source != "remote" {
			t.Fatalf("expected branch %d source %q, got %q", i, "remote", branch.Source)
		}
		if branch.RemoteName != "origin" {
			t.Fatalf("expected branch %d remote name %q, got %q", i, "origin", branch.RemoteName)
		}
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}

func runGitExpectError(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	_, err := cmd.CombinedOutput()
	return err
}
