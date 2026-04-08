package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jeonghyeon-net/baker/internal/config"
	"github.com/jeonghyeon-net/baker/internal/domain"
)

type fakeGitClient struct {
	cloneBareRemoteURL string
	cloneBareRepoPath  string
	cloneBareCalls     int
	cloneBareErr       error
	fetchAllRepoPath   string
	fetchAllCalls      int
	fetchAllErr        error
}

func (f *fakeGitClient) CloneBare(ctx context.Context, remoteURL, repoPath string) error {
	f.cloneBareRemoteURL = remoteURL
	f.cloneBareRepoPath = repoPath
	f.cloneBareCalls++
	return f.cloneBareErr
}

func (f *fakeGitClient) FetchAll(ctx context.Context, repoPath string) error {
	f.fetchAllRepoPath = repoPath
	f.fetchAllCalls++
	return f.fetchAllErr
}

func TestServiceCreateFromRemoteURL(t *testing.T) {
	gitClient := &fakeGitClient{}
	service := Service{
		Git:   gitClient,
		Paths: config.DefaultPaths("/tmp"),
	}

	workspace, err := service.CreateFromRemoteURL(context.Background(), "git@github.com:jeonghyeon-net/baker.git", "jeonghyeon-net-baker")
	if err != nil {
		t.Fatalf("CreateFromRemoteURL() error = %v", err)
	}

	if gitClient.cloneBareCalls != 1 {
		t.Fatalf("CloneBare() call count = %d, want %d", gitClient.cloneBareCalls, 1)
	}
	if gitClient.cloneBareRemoteURL != "git@github.com:jeonghyeon-net/baker.git" {
		t.Fatalf("CloneBare() remoteURL = %q, want %q", gitClient.cloneBareRemoteURL, "git@github.com:jeonghyeon-net/baker.git")
	}
	if gitClient.cloneBareRepoPath != "/tmp/.pi/repositories/jeonghyeon-net-baker" {
		t.Fatalf("CloneBare() repoPath = %q, want %q", gitClient.cloneBareRepoPath, "/tmp/.pi/repositories/jeonghyeon-net-baker")
	}

	if workspace.Name != "jeonghyeon-net-baker" {
		t.Fatalf("Workspace.Name = %q, want %q", workspace.Name, "jeonghyeon-net-baker")
	}
	if workspace.Owner != "jeonghyeon-net" {
		t.Fatalf("Workspace.Owner = %q, want %q", workspace.Owner, "jeonghyeon-net")
	}
	if workspace.Repo != "baker" {
		t.Fatalf("Workspace.Repo = %q, want %q", workspace.Repo, "baker")
	}
	if workspace.RepositoryPath != "/tmp/.pi/repositories/jeonghyeon-net-baker" {
		t.Fatalf("Workspace.RepositoryPath = %q, want %q", workspace.RepositoryPath, "/tmp/.pi/repositories/jeonghyeon-net-baker")
	}
}

func TestServiceCreateFromRemoteURLRejectsEscapingWorkspaceName(t *testing.T) {
	gitClient := &fakeGitClient{}
	service := Service{
		Git:   gitClient,
		Paths: config.DefaultPaths("/tmp"),
	}

	_, err := service.CreateFromRemoteURL(context.Background(), "git@github.com:jeonghyeon-net/baker.git", "../outside")
	if err == nil {
		t.Fatal("CreateFromRemoteURL() error = nil, want error")
	}
	if err.Error() != "invalid workspace name: \"../outside\"" {
		t.Fatalf("CreateFromRemoteURL() error = %q, want %q", err.Error(), "invalid workspace name: \"../outside\"")
	}
	if gitClient.cloneBareCalls != 0 {
		t.Fatalf("CloneBare() call count = %d, want %d", gitClient.cloneBareCalls, 0)
	}
}

func TestServiceCreateFromRemoteURLRejectsRelativeRepositoriesRoot(t *testing.T) {
	gitClient := &fakeGitClient{}
	service := Service{
		Git: gitClient,
		Paths: config.Paths{
			RepositoriesRoot: "repositories",
		},
	}

	_, err := service.CreateFromRemoteURL(context.Background(), "git@github.com:jeonghyeon-net/baker.git", "jeonghyeon-net-baker")
	if err == nil {
		t.Fatal("CreateFromRemoteURL() error = nil, want error")
	}
	if err.Error() != "invalid repositories root: \"repositories\"" {
		t.Fatalf("CreateFromRemoteURL() error = %q, want %q", err.Error(), "invalid repositories root: \"repositories\"")
	}
	if gitClient.cloneBareCalls != 0 {
		t.Fatalf("CloneBare() call count = %d, want %d", gitClient.cloneBareCalls, 0)
	}
}

func TestServiceCreateFromGitHubRepoCopiesDefaultBranch(t *testing.T) {
	gitClient := &fakeGitClient{}
	service := Service{
		Git:   gitClient,
		Paths: config.DefaultPaths("/tmp"),
	}
	repo := domain.GitHubRepo{
		NameWithOwner: "jeonghyeon-net/baker",
		SSHURL:        "git@github.com:jeonghyeon-net/baker.git",
		DefaultBranch: "main",
	}

	workspace, err := service.CreateFromGitHubRepo(context.Background(), repo, "jeonghyeon-net-baker")
	if err != nil {
		t.Fatalf("CreateFromGitHubRepo() error = %v", err)
	}

	if workspace.DefaultBranch != "main" {
		t.Fatalf("Workspace.DefaultBranch = %q, want %q", workspace.DefaultBranch, "main")
	}
}

func TestServiceSyncFetchesAll(t *testing.T) {
	gitClient := &fakeGitClient{}
	service := Service{
		Git:   gitClient,
		Paths: config.DefaultPaths("/tmp"),
	}
	workspace := domain.Workspace{RepositoryPath: "/tmp/.pi/repositories/jeonghyeon-net-baker"}

	if err := service.Sync(context.Background(), workspace); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if gitClient.fetchAllCalls != 1 {
		t.Fatalf("FetchAll() call count = %d, want %d", gitClient.fetchAllCalls, 1)
	}
	if gitClient.fetchAllRepoPath != "/tmp/.pi/repositories/jeonghyeon-net-baker" {
		t.Fatalf("FetchAll() repoPath = %q, want %q", gitClient.fetchAllRepoPath, "/tmp/.pi/repositories/jeonghyeon-net-baker")
	}
}

func TestServiceSyncRejectsRepositoryPathOutsideRepositoriesRoot(t *testing.T) {
	gitClient := &fakeGitClient{}
	service := Service{
		Git:   gitClient,
		Paths: config.DefaultPaths("/tmp"),
	}
	workspace := domain.Workspace{RepositoryPath: "/tmp/outside/jeonghyeon-net-baker"}

	err := service.Sync(context.Background(), workspace)
	if err == nil {
		t.Fatal("Sync() error = nil, want error")
	}
	if err.Error() != "invalid repository path: \"/tmp/outside/jeonghyeon-net-baker\"" {
		t.Fatalf("Sync() error = %q, want %q", err.Error(), "invalid repository path: \"/tmp/outside/jeonghyeon-net-baker\"")
	}
	if gitClient.fetchAllCalls != 0 {
		t.Fatalf("FetchAll() call count = %d, want %d", gitClient.fetchAllCalls, 0)
	}
}

func TestServiceDeleteRemovesRepositoryAndWorkspaceWorktrees(t *testing.T) {
	root := t.TempDir()
	service := Service{
		Git:   &fakeGitClient{},
		Paths: config.DefaultPaths(root),
	}
	workspace := domain.Workspace{
		Name:           "jeonghyeon-net-baker",
		RepositoryPath: filepath.Join(root, ".pi", "repositories", "jeonghyeon-net-baker"),
	}
	worktreePath := filepath.Join(root, ".pi", "worktrees", "jeonghyeon-net-baker", "main")

	if err := os.MkdirAll(workspace.RepositoryPath, 0o755); err != nil {
		t.Fatalf("MkdirAll(repository) error = %v", err)
	}
	if err := os.MkdirAll(worktreePath, 0o755); err != nil {
		t.Fatalf("MkdirAll(worktree) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace.RepositoryPath, "config"), []byte("bare"), 0o644); err != nil {
		t.Fatalf("WriteFile(repository) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktreePath, "README"), []byte("worktree"), 0o644); err != nil {
		t.Fatalf("WriteFile(worktree) error = %v", err)
	}

	if err := service.Delete(workspace); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := os.Stat(workspace.RepositoryPath); !os.IsNotExist(err) {
		t.Fatalf("repository path still exists, err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".pi", "worktrees", "jeonghyeon-net-baker")); !os.IsNotExist(err) {
		t.Fatalf("workspace worktrees path still exists, err = %v", err)
	}
}

func TestServiceDeleteRejectsRepositoryPathOutsideRepositoriesRoot(t *testing.T) {
	service := Service{
		Git:   &fakeGitClient{},
		Paths: config.DefaultPaths("/tmp"),
	}
	workspace := domain.Workspace{
		Name:           "jeonghyeon-net-baker",
		RepositoryPath: "/tmp/outside/jeonghyeon-net-baker",
	}

	err := service.Delete(workspace)
	if err == nil {
		t.Fatal("Delete() error = nil, want error")
	}
	if err.Error() != "invalid repository path: \"/tmp/outside/jeonghyeon-net-baker\"" {
		t.Fatalf("Delete() error = %q, want %q", err.Error(), "invalid repository path: \"/tmp/outside/jeonghyeon-net-baker\"")
	}
}
