package worktree

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

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
	SetBranchUpstream(ctx context.Context, worktreePath, branch, remoteName string) error
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

	path, err := s.worktreePath(workspace.Name, worktreeName)
	if err != nil {
		return CreateResult{}, err
	}
	if err := s.Git.AddExistingBranchWorktree(ctx, workspace.RepositoryPath, branch, path); err != nil {
		return CreateResult{}, err
	}

	remoteName := "origin"
	for _, branchRef := range branches {
		if branchRef.Name == branch && branchRef.RemoteName != "" {
			remoteName = branchRef.RemoteName
			break
		}
	}
	if err := s.Git.SetBranchUpstream(ctx, path, branch, remoteName); err != nil {
		return CreateResult{Path: path, Partial: true}, err
	}

	return CreateResult{Path: path}, nil
}

func (s Service) CreateFromNewBranch(ctx context.Context, workspace domain.Workspace, baseBranch, newBranch, worktreeName string) (CreateResult, error) {
	path, err := s.worktreePath(workspace.Name, worktreeName)
	if err != nil {
		return CreateResult{}, err
	}
	if err := s.Git.AddNewBranchWorktree(ctx, workspace.RepositoryPath, baseBranch, newBranch, path); err != nil {
		return CreateResult{}, err
	}

	if err := s.Git.PushBranch(ctx, path, newBranch); err != nil {
		return CreateResult{Path: path, Partial: true}, err
	}

	return CreateResult{Path: path}, nil
}

func (s Service) Delete(ctx context.Context, workspace domain.Workspace, worktree domain.Worktree, mode DeleteMode, force bool) error {
	if !isValidDeleteMode(mode) {
		return fmt.Errorf("unknown delete mode: %q", mode)
	}

	repositoriesRoot, err := validateRoot(s.Paths.RepositoriesRoot, "repositories root")
	if err != nil {
		return err
	}
	worktreesRoot, err := validateRoot(s.Paths.WorktreesRoot, "worktrees root")
	if err != nil {
		return err
	}
	workspaceName, err := validateName(workspace.Name, "workspace name")
	if err != nil {
		return err
	}
	repositoryPath, err := validatePathWithin(repositoriesRoot, workspace.RepositoryPath, "repository path")
	if err != nil {
		return err
	}
	workspaceWorktreesRoot := filepath.Join(worktreesRoot, workspaceName)
	worktreePath, err := validatePathWithin(workspaceWorktreesRoot, worktree.Path, "worktree path")
	if err != nil {
		return err
	}

	if err := s.Git.RemoveWorktree(ctx, repositoryPath, worktreePath, force); err != nil {
		return err
	}

	switch mode {
	case DeleteModeWorktreeOnly:
		return nil
	case DeleteModeLocalBranch:
		return s.Git.DeleteLocalBranch(ctx, repositoryPath, worktree.BranchName, force)
	case DeleteModeAll:
		if err := s.Git.DeleteLocalBranch(ctx, repositoryPath, worktree.BranchName, force); err != nil {
			return err
		}
		return s.Git.DeleteRemoteBranch(ctx, repositoryPath, worktree.BranchName)
	default:
		return fmt.Errorf("unknown delete mode: %q", mode)
	}
}

func (s Service) worktreePath(workspaceName, worktreeName string) (string, error) {
	worktreesRoot, err := validateRoot(s.Paths.WorktreesRoot, "worktrees root")
	if err != nil {
		return "", err
	}
	workspaceName, err = validateName(workspaceName, "workspace name")
	if err != nil {
		return "", err
	}
	worktreeName, err = validateName(worktreeName, "worktree name")
	if err != nil {
		return "", err
	}

	return filepath.Join(worktreesRoot, workspaceName, worktreeName), nil
}

func isValidDeleteMode(mode DeleteMode) bool {
	switch mode {
	case DeleteModeWorktreeOnly, DeleteModeLocalBranch, DeleteModeAll:
		return true
	default:
		return false
	}
}

func validateRoot(root, label string) (string, error) {
	if root == "" || !filepath.IsAbs(root) {
		return "", fmt.Errorf("invalid %s: %q", label, root)
	}

	return filepath.Clean(root), nil
}

func validateName(name, label string) (string, error) {
	if name == "" || name == "." || name == ".." || filepath.IsAbs(name) || strings.ContainsAny(name, `/\\`) {
		return "", fmt.Errorf("invalid %s: %q", label, name)
	}

	return name, nil
}

func validatePathWithin(root, path, label string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("invalid %s: %q", label, path)
	}

	cleanPath := filepath.Clean(path)
	rel, err := filepath.Rel(root, cleanPath)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid %s: %q", label, path)
	}

	return cleanPath, nil
}
