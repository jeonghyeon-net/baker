package github

import (
	"context"
	"testing"

	"github.com/jeonghyeon-net/baker/internal/domain"
	internalexec "github.com/jeonghyeon-net/baker/internal/exec"
)

type fakeRunner struct {
	result   internalexec.Result
	err      error
	lastName string
	lastArgs []string
}

func (r *fakeRunner) Run(ctx context.Context, name string, args ...string) (internalexec.Result, error) {
	r.lastName = name
	r.lastArgs = append([]string(nil), args...)
	return r.result, r.err
}

func TestClientListRepositories(t *testing.T) {
	runner := &fakeRunner{
		result: internalexec.Result{
			Stdout: `[{"nameWithOwner":"jeonghyeon-net/baker","sshUrl":"git@github.com:jeonghyeon-net/baker.git","defaultBranchRef":{"name":"main"}}]`,
		},
	}
	client := Client{Runner: runner}

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

	expectedArgs := []string{"repo", "list", "--limit", "200", "--json", "nameWithOwner,sshUrl,defaultBranchRef"}
	if runner.lastName != "gh" {
		t.Fatalf("expected command name %q, got %q", "gh", runner.lastName)
	}
	if len(runner.lastArgs) != len(expectedArgs) {
		t.Fatalf("expected args %v, got %v", expectedArgs, runner.lastArgs)
	}
	for i, arg := range expectedArgs {
		if runner.lastArgs[i] != arg {
			t.Fatalf("expected arg %d to be %q, got %q", i, arg, runner.lastArgs[i])
		}
	}
}

func TestClientListRepositoriesUsesConfiguredLimit(t *testing.T) {
	runner := &fakeRunner{
		result: internalexec.Result{Stdout: `[]`},
	}
	client := Client{
		Runner:              runner,
		RepositoryListLimit: 500,
	}

	if _, err := client.ListRepositories(context.Background()); err != nil {
		t.Fatalf("ListRepositories returned error: %v", err)
	}

	expectedArgs := []string{"repo", "list", "--limit", "500", "--json", "nameWithOwner,sshUrl,defaultBranchRef"}
	if len(runner.lastArgs) != len(expectedArgs) {
		t.Fatalf("expected args %v, got %v", expectedArgs, runner.lastArgs)
	}
	for i, arg := range expectedArgs {
		if runner.lastArgs[i] != arg {
			t.Fatalf("expected arg %d to be %q, got %q", i, arg, runner.lastArgs[i])
		}
	}
}
