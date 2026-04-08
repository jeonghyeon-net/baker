package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeonghyeon-net/baker/internal/config"
	"github.com/jeonghyeon-net/baker/internal/domain"
)

type GitClient interface {
	CloneBare(ctx context.Context, remoteURL, repoPath string) error
	FetchAll(ctx context.Context, repoPath string) error
}

type Service struct {
	Git   GitClient
	Paths config.Paths
}

func (s Service) CreateFromRemoteURL(ctx context.Context, remoteURL, workspaceName string) (domain.Workspace, error) {
	repositoriesRoot, err := validateRepositoriesRoot(s.Paths.RepositoriesRoot)
	if err != nil {
		return domain.Workspace{}, err
	}

	workspaceName, err = validateWorkspaceName(workspaceName)
	if err != nil {
		return domain.Workspace{}, err
	}

	owner, repo, err := parseGitHubSSHRemote(remoteURL)
	if err != nil {
		return domain.Workspace{}, err
	}

	repositoryPath := filepath.Join(repositoriesRoot, workspaceName)
	workspace := domain.Workspace{
		Name:           workspaceName,
		RemoteURL:      remoteURL,
		Owner:          owner,
		Repo:           repo,
		RepositoryPath: repositoryPath,
	}

	if err := s.Git.CloneBare(ctx, remoteURL, repositoryPath); err != nil {
		return domain.Workspace{}, err
	}

	return workspace, nil
}

func (s Service) CreateFromGitHubRepo(ctx context.Context, repo domain.GitHubRepo, workspaceName string) (domain.Workspace, error) {
	workspace, err := s.CreateFromRemoteURL(ctx, repo.SSHURL, workspaceName)
	if err != nil {
		return domain.Workspace{}, err
	}

	workspace.DefaultBranch = repo.DefaultBranch
	return workspace, nil
}

func (s Service) Sync(ctx context.Context, workspace domain.Workspace) error {
	repositoriesRoot, err := validateRepositoriesRoot(s.Paths.RepositoriesRoot)
	if err != nil {
		return err
	}

	repositoryPath, err := validateRepositoryPath(repositoriesRoot, workspace.RepositoryPath)
	if err != nil {
		return err
	}

	return s.Git.FetchAll(ctx, repositoryPath)
}

func (s Service) Delete(workspace domain.Workspace) error {
	repositoriesRoot, err := validateRepositoriesRoot(s.Paths.RepositoriesRoot)
	if err != nil {
		return err
	}
	worktreesRoot, err := validateRoot(s.Paths.WorktreesRoot, "worktrees root")
	if err != nil {
		return err
	}
	workspaceName, err := validateWorkspaceName(workspace.Name)
	if err != nil {
		return err
	}
	repositoryPath, err := validateRepositoryPath(repositoriesRoot, workspace.RepositoryPath)
	if err != nil {
		return err
	}
	workspaceWorktreesPath, err := validateManagedPath(worktreesRoot, filepath.Join(worktreesRoot, workspaceName), "workspace worktrees path")
	if err != nil {
		return err
	}

	if err := os.RemoveAll(workspaceWorktreesPath); err != nil {
		return err
	}
	if err := os.RemoveAll(repositoryPath); err != nil {
		return err
	}
	return nil
}

func validateRepositoriesRoot(root string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("invalid repositories root: %q", root)
	}

	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("invalid repositories root: %q", root)
	}

	return filepath.Clean(root), nil
}

func validateRepositoryPath(repositoriesRoot, repositoryPath string) (string, error) {
	if !filepath.IsAbs(repositoryPath) {
		return "", fmt.Errorf("invalid repository path: %q", repositoryPath)
	}

	cleanRepositoryPath := filepath.Clean(repositoryPath)
	rel, err := filepath.Rel(repositoriesRoot, cleanRepositoryPath)
	if err != nil {
		return "", fmt.Errorf("invalid repository path: %q", repositoryPath)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid repository path: %q", repositoryPath)
	}

	return cleanRepositoryPath, nil
}

func validateRoot(root, label string) (string, error) {
	if root == "" || !filepath.IsAbs(root) {
		return "", fmt.Errorf("invalid %s: %q", label, root)
	}
	return filepath.Clean(root), nil
}

func validateManagedPath(root, path, label string) (string, error) {
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

func validateWorkspaceName(workspaceName string) (string, error) {
	if workspaceName == "" || workspaceName == "." || workspaceName == ".." {
		return "", fmt.Errorf("invalid workspace name: %q", workspaceName)
	}

	if filepath.IsAbs(workspaceName) || strings.ContainsAny(workspaceName, `/\\`) {
		return "", fmt.Errorf("invalid workspace name: %q", workspaceName)
	}

	return workspaceName, nil
}

func parseGitHubSSHRemote(remoteURL string) (owner, repo string, err error) {
	path, ok := strings.CutPrefix(remoteURL, "git@github.com:")
	if !ok {
		return "", "", fmt.Errorf("invalid GitHub SSH remote: %q", remoteURL)
	}

	name, ok := strings.CutSuffix(path, ".git")
	if !ok {
		return "", "", fmt.Errorf("invalid GitHub SSH remote: %q", remoteURL)
	}

	owner, repo, ok = strings.Cut(name, "/")
	if !ok || owner == "" || repo == "" || strings.Contains(repo, "/") {
		return "", "", fmt.Errorf("invalid GitHub SSH remote: %q", remoteURL)
	}

	return owner, repo, nil
}
