package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
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

	if len(branches) != 1 {
		t.Fatalf("expected 1 branch, got %d: %#v", len(branches), branches)
	}

	branch := branches[0]
	if branch.Name != "main" {
		t.Fatalf("expected branch name %q, got %q", "main", branch.Name)
	}
	if branch.Source != "remote" {
		t.Fatalf("expected branch source %q, got %q", "remote", branch.Source)
	}
	if branch.RemoteName != "origin" {
		t.Fatalf("expected remote name %q, got %q", "origin", branch.RemoteName)
	}
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
