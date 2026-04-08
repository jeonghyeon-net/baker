package worktree

import (
	"context"
	"errors"
	"testing"

	"github.com/jeonghyeon-net/baker/internal/config"
	"github.com/jeonghyeon-net/baker/internal/domain"
)

type fakeGitClient struct {
	addExistingRepoPath string
	addExistingBranch   string
	addExistingPath     string
	addExistingCalls    int
	addExistingErr      error

	addNewRepoPath string
	addNewBase     string
	addNewBranch   string
	addNewPath     string
	addNewCalls    int
	addNewErr      error

	pushWorktreePath string
	pushBranch       string
	pushCalls        int
	pushErr          error

	removeRepoPath string
	removePath     string
	removeForce    bool
	removeCalls    int
	removeErr      error

	deleteLocalRepoPath string
	deleteLocalBranch   string
	deleteLocalForce    bool
	deleteLocalCalls    int
	deleteLocalErr      error

	deleteRemoteRepoPath string
	deleteRemoteBranch   string
	deleteRemoteCalls    int
	deleteRemoteErr      error
}

func (f *fakeGitClient) AddExistingBranchWorktree(ctx context.Context, repoPath, branch, worktreePath string) error {
	f.addExistingRepoPath = repoPath
	f.addExistingBranch = branch
	f.addExistingPath = worktreePath
	f.addExistingCalls++
	return f.addExistingErr
}

func (f *fakeGitClient) AddNewBranchWorktree(ctx context.Context, repoPath, baseBranch, newBranch, worktreePath string) error {
	f.addNewRepoPath = repoPath
	f.addNewBase = baseBranch
	f.addNewBranch = newBranch
	f.addNewPath = worktreePath
	f.addNewCalls++
	return f.addNewErr
}

func (f *fakeGitClient) PushBranch(ctx context.Context, worktreePath, branch string) error {
	f.pushWorktreePath = worktreePath
	f.pushBranch = branch
	f.pushCalls++
	return f.pushErr
}

func (f *fakeGitClient) RemoveWorktree(ctx context.Context, repoPath, worktreePath string, force bool) error {
	f.removeRepoPath = repoPath
	f.removePath = worktreePath
	f.removeForce = force
	f.removeCalls++
	return f.removeErr
}

func (f *fakeGitClient) DeleteLocalBranch(ctx context.Context, repoPath, branch string, force bool) error {
	f.deleteLocalRepoPath = repoPath
	f.deleteLocalBranch = branch
	f.deleteLocalForce = force
	f.deleteLocalCalls++
	return f.deleteLocalErr
}

func (f *fakeGitClient) DeleteRemoteBranch(ctx context.Context, repoPath, branch string) error {
	f.deleteRemoteRepoPath = repoPath
	f.deleteRemoteBranch = branch
	f.deleteRemoteCalls++
	return f.deleteRemoteErr
}

func TestServiceCreateFromExistingBranchBlocksDuplicateActiveWorktree(t *testing.T) {
	gitClient := &fakeGitClient{}
	service := Service{
		Git: gitClient,
		Paths: config.Paths{
			WorktreesRoot: "/tmp/.pi/worktrees",
		},
	}
	workspace := domain.Workspace{
		Name:           "baker",
		RepositoryPath: "/tmp/.pi/repositories/baker",
	}
	branches := []domain.BranchRef{{
		Name:              "main",
		HasActiveWorktree: true,
	}}

	_, err := service.CreateFromExistingBranch(context.Background(), workspace, branches, "main", "main")
	if err == nil {
		t.Fatal("CreateFromExistingBranch() error = nil, want error")
	}
	if err.Error() != "branch main already has an active worktree" {
		t.Fatalf("CreateFromExistingBranch() error = %q, want %q", err.Error(), "branch main already has an active worktree")
	}
	if gitClient.addExistingCalls != 0 {
		t.Fatalf("AddExistingBranchWorktree() call count = %d, want %d", gitClient.addExistingCalls, 0)
	}
}

func TestServiceCreateFromExistingBranchRejectsEscapingWorktreeName(t *testing.T) {
	gitClient := &fakeGitClient{}
	service := Service{
		Git: gitClient,
		Paths: config.Paths{
			WorktreesRoot: "/tmp/.pi/worktrees",
		},
	}
	workspace := domain.Workspace{
		Name:           "baker",
		RepositoryPath: "/tmp/.pi/repositories/baker",
	}

	_, err := service.CreateFromExistingBranch(context.Background(), workspace, nil, "main", "../escape")
	if err == nil {
		t.Fatal("CreateFromExistingBranch() error = nil, want error")
	}
	if err.Error() != "invalid worktree name: \"../escape\"" {
		t.Fatalf("CreateFromExistingBranch() error = %q, want %q", err.Error(), "invalid worktree name: \"../escape\"")
	}
	if gitClient.addExistingCalls != 0 {
		t.Fatalf("AddExistingBranchWorktree() call count = %d, want %d", gitClient.addExistingCalls, 0)
	}
}

func TestServiceCreateFromNewBranchReturnsPartialResultWhenPushFails(t *testing.T) {
	pushErr := errors.New("push failed")
	gitClient := &fakeGitClient{pushErr: pushErr}
	service := Service{
		Git: gitClient,
		Paths: config.Paths{
			WorktreesRoot: "/tmp/.pi/worktrees",
		},
	}
	workspace := domain.Workspace{
		Name:           "baker",
		RepositoryPath: "/tmp/.pi/repositories/baker",
	}

	result, err := service.CreateFromNewBranch(context.Background(), workspace, "main", "feature/login", "feature-login")
	if !errors.Is(err, pushErr) {
		t.Fatalf("CreateFromNewBranch() error = %v, want %v", err, pushErr)
	}
	if result != (CreateResult{Path: "/tmp/.pi/worktrees/baker/feature-login", Partial: true}) {
		t.Fatalf("CreateFromNewBranch() result = %#v, want %#v", result, CreateResult{Path: "/tmp/.pi/worktrees/baker/feature-login", Partial: true})
	}
	if gitClient.addNewCalls != 1 {
		t.Fatalf("AddNewBranchWorktree() call count = %d, want %d", gitClient.addNewCalls, 1)
	}
	if gitClient.pushCalls != 1 {
		t.Fatalf("PushBranch() call count = %d, want %d", gitClient.pushCalls, 1)
	}
}

func TestServiceDeleteRemovesWorktreeAndDeletesLocalAndRemoteBranch(t *testing.T) {
	gitClient := &fakeGitClient{}
	service := Service{
		Git: gitClient,
		Paths: config.Paths{
			RepositoriesRoot: "/tmp/.pi/repositories",
			WorktreesRoot:    "/tmp/.pi/worktrees",
		},
	}
	workspace := domain.Workspace{
		Name:           "baker",
		RepositoryPath: "/tmp/.pi/repositories/baker",
	}
	worktree := domain.Worktree{
		Path:       "/tmp/.pi/worktrees/baker/feature-login",
		BranchName: "feature/login",
	}

	err := service.Delete(context.Background(), workspace, worktree, DeleteModeAll, true)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gitClient.removeCalls != 1 {
		t.Fatalf("RemoveWorktree() call count = %d, want %d", gitClient.removeCalls, 1)
	}
	if gitClient.removeRepoPath != "/tmp/.pi/repositories/baker" {
		t.Fatalf("RemoveWorktree() repoPath = %q, want %q", gitClient.removeRepoPath, "/tmp/.pi/repositories/baker")
	}
	if gitClient.removePath != "/tmp/.pi/worktrees/baker/feature-login" {
		t.Fatalf("RemoveWorktree() worktreePath = %q, want %q", gitClient.removePath, "/tmp/.pi/worktrees/baker/feature-login")
	}
	if !gitClient.removeForce {
		t.Fatal("RemoveWorktree() force = false, want true")
	}
	if gitClient.deleteLocalCalls != 1 {
		t.Fatalf("DeleteLocalBranch() call count = %d, want %d", gitClient.deleteLocalCalls, 1)
	}
	if gitClient.deleteLocalRepoPath != "/tmp/.pi/repositories/baker" {
		t.Fatalf("DeleteLocalBranch() repoPath = %q, want %q", gitClient.deleteLocalRepoPath, "/tmp/.pi/repositories/baker")
	}
	if gitClient.deleteLocalBranch != "feature/login" {
		t.Fatalf("DeleteLocalBranch() branch = %q, want %q", gitClient.deleteLocalBranch, "feature/login")
	}
	if !gitClient.deleteLocalForce {
		t.Fatal("DeleteLocalBranch() force = false, want true")
	}
	if gitClient.deleteRemoteCalls != 1 {
		t.Fatalf("DeleteRemoteBranch() call count = %d, want %d", gitClient.deleteRemoteCalls, 1)
	}
	if gitClient.deleteRemoteRepoPath != "/tmp/.pi/repositories/baker" {
		t.Fatalf("DeleteRemoteBranch() repoPath = %q, want %q", gitClient.deleteRemoteRepoPath, "/tmp/.pi/repositories/baker")
	}
	if gitClient.deleteRemoteBranch != "feature/login" {
		t.Fatalf("DeleteRemoteBranch() branch = %q, want %q", gitClient.deleteRemoteBranch, "feature/login")
	}
}

func TestServiceDeleteRejectsEscapingWorktreePath(t *testing.T) {
	gitClient := &fakeGitClient{}
	service := Service{
		Git: gitClient,
		Paths: config.Paths{
			RepositoriesRoot: "/tmp/.pi/repositories",
			WorktreesRoot:    "/tmp/.pi/worktrees",
		},
	}
	workspace := domain.Workspace{
		Name:           "baker",
		RepositoryPath: "/tmp/.pi/repositories/baker",
	}
	worktree := domain.Worktree{
		Path:       "/tmp/.pi/worktrees/other/feature-login",
		BranchName: "feature/login",
	}

	err := service.Delete(context.Background(), workspace, worktree, DeleteModeAll, true)
	if err == nil {
		t.Fatal("Delete() error = nil, want error")
	}
	if err.Error() != "invalid worktree path: \"/tmp/.pi/worktrees/other/feature-login\"" {
		t.Fatalf("Delete() error = %q, want %q", err.Error(), "invalid worktree path: \"/tmp/.pi/worktrees/other/feature-login\"")
	}
	if gitClient.removeCalls != 0 {
		t.Fatalf("RemoveWorktree() call count = %d, want %d", gitClient.removeCalls, 0)
	}
}

func TestServiceDeleteRejectsUnknownMode(t *testing.T) {
	gitClient := &fakeGitClient{}
	service := Service{
		Git: gitClient,
		Paths: config.Paths{
			RepositoriesRoot: "/tmp/.pi/repositories",
			WorktreesRoot:    "/tmp/.pi/worktrees",
		},
	}
	workspace := domain.Workspace{
		Name:           "baker",
		RepositoryPath: "/tmp/.pi/repositories/baker",
	}
	worktree := domain.Worktree{
		Path:       "/tmp/.pi/worktrees/baker/feature-login",
		BranchName: "feature/login",
	}

	err := service.Delete(context.Background(), workspace, worktree, DeleteMode("unknown"), true)
	if err == nil {
		t.Fatal("Delete() error = nil, want error")
	}
	if err.Error() != "unknown delete mode: \"unknown\"" {
		t.Fatalf("Delete() error = %q, want %q", err.Error(), "unknown delete mode: \"unknown\"")
	}
	if gitClient.removeCalls != 0 {
		t.Fatalf("RemoveWorktree() call count = %d, want %d", gitClient.removeCalls, 0)
	}
}
