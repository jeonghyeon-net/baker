package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeonghyeon-net/baker/internal/config"
	"github.com/jeonghyeon-net/baker/internal/domain"
	internalexec "github.com/jeonghyeon-net/baker/internal/exec"
	bakergit "github.com/jeonghyeon-net/baker/internal/git"
)

func TestEnsureProcessOutsidePathMovesToHomeWhenInsideTarget(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	target := filepath.Join(t.TempDir(), "worktree")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	if err := os.Chdir(target); err != nil {
		t.Fatalf("Chdir(target) error = %v", err)
	}
	fallbackPath, err := ensureProcessOutsidePath(target)
	if err != nil {
		t.Fatalf("ensureProcessOutsidePath() error = %v", err)
	}
	if canonicalPath(fallbackPath) != canonicalPath(home) {
		t.Fatalf("fallbackPath = %q, want %q", fallbackPath, home)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() after ensureProcessOutsidePath error = %v", err)
	}
	if canonicalPath(wd) != canonicalPath(home) {
		t.Fatalf("working directory = %q, want %q", wd, home)
	}
}

func TestEnsureProcessOutsidePathDoesNothingWhenOutsideTarget(t *testing.T) {
	outside := t.TempDir()
	target := filepath.Join(t.TempDir(), "worktree")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	if err := os.Chdir(outside); err != nil {
		t.Fatalf("Chdir(outside) error = %v", err)
	}
	fallbackPath, err := ensureProcessOutsidePath(target)
	if err != nil {
		t.Fatalf("ensureProcessOutsidePath() error = %v", err)
	}
	if fallbackPath != "" {
		t.Fatalf("fallbackPath = %q, want empty", fallbackPath)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() after ensureProcessOutsidePath error = %v", err)
	}
	if canonicalPath(wd) != canonicalPath(outside) {
		t.Fatalf("working directory = %q, want %q", wd, outside)
	}
}

func TestCreateWorktreeModeOptionsPrioritizeNewBranch(t *testing.T) {
	got := createWorktreeModeOptions()
	want := []optionChoice{{Label: "새 브랜치 만들어 생성", Value: "new"}, {Label: "기존 브랜치로 생성", Value: "existing"}}

	if len(got) != len(want) {
		t.Fatalf("len(createWorktreeModeOptions()) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("createWorktreeModeOptions()[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestPrioritizedBranchNames(t *testing.T) {
	tests := []struct {
		name     string
		branches []string
		want     []string
	}{
		{name: "moves development production and main to front", branches: []string{"release", "main", "production", "feature/login", "development"}, want: []string{"development", "production", "main", "release", "feature/login"}},
		{name: "keeps missing priority branches absent", branches: []string{"release", "feature/login"}, want: []string{"release", "feature/login"}},
		{name: "keeps only present priority branches", branches: []string{"release", "main", "feature/login"}, want: []string{"main", "release", "feature/login"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prioritizedBranchNames(tt.branches)
			if len(got) != len(tt.want) {
				t.Fatalf("len(prioritizedBranchNames()) = %d, want %d (%#v)", len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("prioritizedBranchNames()[%d] = %q, want %q (%#v)", i, got[i], tt.want[i], got)
				}
			}
		})
	}
}

func TestPullRequestStatusLabel(t *testing.T) {
	tests := []struct {
		name string
		pr   domain.GitHubPullRequest
		want string
	}{
		{name: "draft", pr: domain.GitHubPullRequest{IsDraft: true}, want: "초안"},
		{name: "approved", pr: domain.GitHubPullRequest{ReviewDecision: "APPROVED"}, want: "승인"},
		{name: "changes requested", pr: domain.GitHubPullRequest{ReviewDecision: "CHANGES_REQUESTED"}, want: "수정 요청"},
		{name: "review required", pr: domain.GitHubPullRequest{ReviewDecision: "REVIEW_REQUIRED"}, want: "리뷰 대기"},
		{name: "unknown", pr: domain.GitHubPullRequest{ReviewDecision: "COMMENTED"}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pullRequestStatusLabel(tt.pr); got != tt.want {
				t.Fatalf("pullRequestStatusLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadWorktreeItemsDoesNotFetchRemoteStatusSynchronously(t *testing.T) {
	root := t.TempDir()
	paths := config.DefaultPaths(root)
	repoPath := filepath.Join(paths.RepositoriesRoot, "baker")
	worktreePath := filepath.Join(paths.WorktreesRoot, "baker", "feature-login")

	runner := &stubRunner{responses: map[string]stubRunnerResponse{
		"git --git-dir " + repoPath + " worktree list --porcelain": {
			result: internalexec.Result{Stdout: "worktree " + worktreePath + "\nHEAD deadbeef\nbranch refs/heads/feature/login\n"},
		},
	}}

	items, err := loadWorktreeItems(context.Background(), paths, config.Registry{Workspaces: []domain.Workspace{{Name: "baker", RepositoryPath: repoPath}}}, bakergit.Client{Runner: runner})
	if err != nil {
		t.Fatalf("loadWorktreeItems() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2 (%#v)", len(items), items)
	}
	if items[1].MissingRemote {
		t.Fatalf("items[1].MissingRemote = %v, want false before async refresh (%#v)", items[1].MissingRemote, items[1])
	}
}

func TestLoadMissingRemoteBranchesReturnsDeletedBranches(t *testing.T) {
	root := t.TempDir()
	paths := config.DefaultPaths(root)
	repoPath := filepath.Join(paths.RepositoriesRoot, "baker")
	branchPaths := map[string]string{
		"feature/login": filepath.Join(paths.WorktreesRoot, "baker", "feature-login"),
		"main":          filepath.Join(paths.WorktreesRoot, "baker", "main"),
	}

	runner := &stubRunner{responses: map[string]stubRunnerResponse{
		"git --git-dir " + repoPath + " remote": {
			result: internalexec.Result{Stdout: "origin\n"},
		},
		"git --git-dir " + repoPath + " fetch --prune --force --filter=blob:none origin": {},
		"git --git-dir " + repoPath + " for-each-ref --format=%(refname:lstrip=3)\tremote\torigin refs/remotes/origin": {
			result: internalexec.Result{Stdout: "main\tremote\torigin\n"},
		},
	}}

	missing, ok := loadMissingRemoteBranches(context.Background(), bakergit.Client{Runner: runner}, repoPath, branchPaths)
	if !ok {
		t.Fatal("loadMissingRemoteBranches() ok = false, want true")
	}
	if len(missing) != 1 || missing[0] != "feature/login" {
		t.Fatalf("loadMissingRemoteBranches() = %#v, want %#v", missing, []string{"feature/login"})
	}
	if !containsCall(runner.calls, "git --git-dir "+repoPath+" fetch --prune --force --filter=blob:none origin") {
		t.Fatalf("expected fetch call, got %#v", runner.calls)
	}
}

func TestLoadMissingRemoteBranchesReturnsFalseWhenRefreshFails(t *testing.T) {
	root := t.TempDir()
	paths := config.DefaultPaths(root)
	repoPath := filepath.Join(paths.RepositoriesRoot, "baker")
	branchPaths := map[string]string{"feature/login": filepath.Join(paths.WorktreesRoot, "baker", "feature-login")}

	runner := &stubRunner{responses: map[string]stubRunnerResponse{
		"git --git-dir " + repoPath + " remote": {
			result: internalexec.Result{Stdout: "origin\n"},
		},
		"git --git-dir " + repoPath + " fetch --prune --force --filter=blob:none origin": {
			result: internalexec.Result{Stderr: "fatal: network timeout"},
			err:    errors.New("fetch failed"),
		},
	}}

	missing, ok := loadMissingRemoteBranches(context.Background(), bakergit.Client{Runner: runner}, repoPath, branchPaths)
	if ok {
		t.Fatalf("loadMissingRemoteBranches() ok = true, want false (missing=%#v)", missing)
	}
}

type stubRunnerResponse struct {
	result internalexec.Result
	err    error
}

type stubRunner struct {
	responses map[string]stubRunnerResponse
	calls     []string
}

func (s *stubRunner) Run(ctx context.Context, name string, args ...string) (internalexec.Result, error) {
	key := name + " " + strings.Join(args, " ")
	s.calls = append(s.calls, key)
	response, ok := s.responses[key]
	if !ok {
		return internalexec.Result{}, fmt.Errorf("unexpected command: %s", key)
	}
	return response.result, response.err
}

func containsCall(calls []string, want string) bool {
	for _, call := range calls {
		if call == want {
			return true
		}
	}
	return false
}
