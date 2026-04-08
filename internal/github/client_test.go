package github

import (
	"context"
	"testing"

	"github.com/jeonghyeon-net/baker/internal/domain"
	internalexec "github.com/jeonghyeon-net/baker/internal/exec"
)

type fakeRunner struct {
	result internalexec.Result
	err    error
}

func (r fakeRunner) Run(ctx context.Context, name string, args ...string) (internalexec.Result, error) {
	return r.result, r.err
}

func TestClientListRepositories(t *testing.T) {
	client := Client{
		Runner: fakeRunner{
			result: internalexec.Result{
				Stdout: `[{"nameWithOwner":"jeonghyeon-net/baker","sshUrl":"git@github.com:jeonghyeon-net/baker.git","defaultBranchRef":{"name":"main"}}]`,
			},
		},
	}

	repos, err := client.ListRepositories(context.Background())
	if err != nil {
		t.Fatalf("ListRepositories returned error: %v", err)
	}

	expected := []domain.GitHubRepo{
		{
			NameWithOwner: "jeonghyeon-net/baker",
			SSHURL:        "git@github.com:jeonghyeon-net/baker.git",
			DefaultBranch: "main",
		},
	}

	if len(repos) != len(expected) {
		t.Fatalf("expected %d repos, got %d", len(expected), len(repos))
	}

	if repos[0] != expected[0] {
		t.Fatalf("expected repo %#v, got %#v", expected[0], repos[0])
	}
}
