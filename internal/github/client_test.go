package github

import (
	"context"
	"testing"

	"github.com/jeonghyeon-net/baker/internal/domain"
	internalexec "github.com/jeonghyeon-net/baker/internal/exec"
)

type fakeRunnerCall struct {
	name string
	args []string
}

type fakeRunner struct {
	results []internalexec.Result
	err     error
	calls   []fakeRunnerCall
}

func (r *fakeRunner) Run(ctx context.Context, name string, args ...string) (internalexec.Result, error) {
	r.calls = append(r.calls, fakeRunnerCall{name: name, args: append([]string(nil), args...)})
	if r.err != nil {
		return internalexec.Result{}, r.err
	}
	if len(r.results) == 0 {
		return internalexec.Result{}, nil
	}
	result := r.results[0]
	r.results = r.results[1:]
	return result, nil
}

func TestClientListRepositoriesIncludesUserAndOrganizations(t *testing.T) {
	runner := &fakeRunner{results: []internalexec.Result{
		{Stdout: `{"login":"jeonghyeon-net"}`},
		{Stdout: `[{"login":"creatrip"},{"login":"platform-team"}]`},
		{Stdout: `[{"nameWithOwner":"jeonghyeon-net/baker","sshUrl":"git@github.com:jeonghyeon-net/baker.git","defaultBranchRef":{"name":"main"}}]`},
		{Stdout: `[{"nameWithOwner":"creatrip/admin","sshUrl":"git@github.com:creatrip/admin.git","defaultBranchRef":{"name":"main"}}]`},
		{Stdout: `[]`},
	}}
	client := Client{Runner: runner}

	repos, err := client.ListRepositories(context.Background())
	if err != nil {
		t.Fatalf("ListRepositories returned error: %v", err)
	}

	expected := []domain.GitHubRepo{
		{NameWithOwner: "jeonghyeon-net/baker", SSHURL: "git@github.com:jeonghyeon-net/baker.git", DefaultBranch: "main"},
		{NameWithOwner: "creatrip/admin", SSHURL: "git@github.com:creatrip/admin.git", DefaultBranch: "main"},
	}
	if len(repos) != len(expected) {
		t.Fatalf("expected %d repos, got %d (%#v)", len(expected), len(repos), repos)
	}
	for i := range expected {
		if repos[i] != expected[i] {
			t.Fatalf("expected repo %#v, got %#v", expected[i], repos[i])
		}
	}

	expectedCalls := []fakeRunnerCall{
		{name: "gh", args: []string{"api", "user"}},
		{name: "gh", args: []string{"api", "user/orgs", "--paginate"}},
		{name: "gh", args: []string{"repo", "list", "jeonghyeon-net", "--limit", "200", "--json", "nameWithOwner,sshUrl,defaultBranchRef"}},
		{name: "gh", args: []string{"repo", "list", "creatrip", "--limit", "200", "--json", "nameWithOwner,sshUrl,defaultBranchRef"}},
		{name: "gh", args: []string{"repo", "list", "platform-team", "--limit", "200", "--json", "nameWithOwner,sshUrl,defaultBranchRef"}},
	}
	if len(runner.calls) != len(expectedCalls) {
		t.Fatalf("expected %d calls, got %d (%#v)", len(expectedCalls), len(runner.calls), runner.calls)
	}
	for i := range expectedCalls {
		if runner.calls[i].name != expectedCalls[i].name {
			t.Fatalf("call %d name = %q, want %q", i, runner.calls[i].name, expectedCalls[i].name)
		}
		if len(runner.calls[i].args) != len(expectedCalls[i].args) {
			t.Fatalf("call %d args = %v, want %v", i, runner.calls[i].args, expectedCalls[i].args)
		}
		for j := range expectedCalls[i].args {
			if runner.calls[i].args[j] != expectedCalls[i].args[j] {
				t.Fatalf("call %d arg %d = %q, want %q", i, j, runner.calls[i].args[j], expectedCalls[i].args[j])
			}
		}
	}
}

func TestClientListRepositoriesUsesConfiguredLimit(t *testing.T) {
	runner := &fakeRunner{results: []internalexec.Result{
		{Stdout: `{"login":"jeonghyeon-net"}`},
		{Stdout: `[]`},
		{Stdout: `[]`},
	}}
	client := Client{Runner: runner, RepositoryListLimit: 500}

	if _, err := client.ListRepositories(context.Background()); err != nil {
		t.Fatalf("ListRepositories returned error: %v", err)
	}

	expectedArgs := []string{"repo", "list", "jeonghyeon-net", "--limit", "500", "--json", "nameWithOwner,sshUrl,defaultBranchRef"}
	last := runner.calls[len(runner.calls)-1]
	if len(last.args) != len(expectedArgs) {
		t.Fatalf("expected args %v, got %v", expectedArgs, last.args)
	}
	for i, arg := range expectedArgs {
		if last.args[i] != arg {
			t.Fatalf("expected arg %d to be %q, got %q", i, arg, last.args[i])
		}
	}
}
