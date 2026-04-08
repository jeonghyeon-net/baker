package workspace

import (
	"context"
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
	service := Service{Git: gitClient}
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
