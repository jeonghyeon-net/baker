package workspace

import (
	"context"
	"fmt"
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
	owner, repo, err := parseGitHubSSHRemote(remoteURL)
	if err != nil {
		return domain.Workspace{}, err
	}

	repositoryPath := filepath.Join(s.Paths.RepositoriesRoot, workspaceName)
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
	return s.Git.FetchAll(ctx, workspace.RepositoryPath)
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
