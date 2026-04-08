package worktree

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/jeonghyeon-net/baker/internal/config"
	"github.com/jeonghyeon-net/baker/internal/domain"
)

type DeleteMode string

const (
	DeleteModeWorktreeOnly DeleteMode = "worktree-only"
	DeleteModeLocalBranch  DeleteMode = "worktree-and-local-branch"
	DeleteModeAll          DeleteMode = "worktree-local-and-remote-branch"
)

type GitClient interface {
	AddExistingBranchWorktree(ctx context.Context, repoPath, branch, worktreePath string) error
	AddNewBranchWorktree(ctx context.Context, repoPath, baseBranch, newBranch, worktreePath string) error
	PushBranch(ctx context.Context, worktreePath, branch string) error
	RemoveWorktree(ctx context.Context, repoPath, worktreePath string, force bool) error
	DeleteLocalBranch(ctx context.Context, repoPath, branch string, force bool) error
	DeleteRemoteBranch(ctx context.Context, repoPath, branch string) error
}

type CreateResult struct {
	Path    string
	Partial bool
}

type Service struct {
	Git   GitClient
	Paths config.Paths
}

func (s Service) CreateFromExistingBranch(ctx context.Context, workspace domain.Workspace, branches []domain.BranchRef, branch, worktreeName string) (CreateResult, error) {
	for _, branchRef := range branches {
		if branchRef.Name == branch && branchRef.HasActiveWorktree {
			return CreateResult{}, fmt.Errorf("branch %s already has an active worktree", branch)
		}
	}

	path := s.worktreePath(workspace, worktreeName)
	if err := s.Git.AddExistingBranchWorktree(ctx, workspace.RepositoryPath, branch, path); err != nil {
		return CreateResult{}, err
	}

	return CreateResult{Path: path}, nil
}

func (s Service) CreateFromNewBranch(ctx context.Context, workspace domain.Workspace, baseBranch, newBranch, worktreeName string) (CreateResult, error) {
	path := s.worktreePath(workspace, worktreeName)
	if err := s.Git.AddNewBranchWorktree(ctx, workspace.RepositoryPath, baseBranch, newBranch, path); err != nil {
		return CreateResult{}, err
	}

	if err := s.Git.PushBranch(ctx, path, newBranch); err != nil {
		return CreateResult{Path: path, Partial: true}, err
	}

	return CreateResult{Path: path}, nil
}

func (s Service) Delete(ctx context.Context, workspace domain.Workspace, worktree domain.Worktree, mode DeleteMode, force bool) error {
	if err := s.Git.RemoveWorktree(ctx, workspace.RepositoryPath, worktree.Path, force); err != nil {
		return err
	}

	switch mode {
	case DeleteModeLocalBranch:
		return s.Git.DeleteLocalBranch(ctx, workspace.RepositoryPath, worktree.BranchName, force)
	case DeleteModeAll:
		if err := s.Git.DeleteLocalBranch(ctx, workspace.RepositoryPath, worktree.BranchName, force); err != nil {
			return err
		}
		return s.Git.DeleteRemoteBranch(ctx, workspace.RepositoryPath, worktree.BranchName)
	default:
		return nil
	}
}

func (s Service) worktreePath(workspace domain.Workspace, worktreeName string) string {
	return filepath.Join(s.Paths.WorktreesRoot, workspace.Name, worktreeName)
}
