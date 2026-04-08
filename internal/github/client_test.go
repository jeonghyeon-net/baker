package github

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/jeonghyeon-net/baker/internal/domain"
	internalexec "github.com/jeonghyeon-net/baker/internal/exec"
)

type fakeRunnerCall struct {
	name string
	args []string
}

type fakeRunner struct {
	mu      sync.Mutex
	results []internalexec.Result
	lookup  map[string]internalexec.Result
	err     error
	calls   []fakeRunnerCall
}

func (r *fakeRunner) Run(ctx context.Context, name string, args ...string) (internalexec.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls = append(r.calls, fakeRunnerCall{name: name, args: append([]string(nil), args...)})
	if r.err != nil {
		return internalexec.Result{}, r.err
	}
	if r.lookup != nil {
		if result, ok := r.lookup[name+"\x00"+strings.Join(args, "\x00")]; ok {
			return result, nil
		}
	}
	if len(r.results) == 0 {
		return internalexec.Result{}, nil
	}
	result := r.results[0]
	r.results = r.results[1:]
	return result, nil
}

func TestClientListOwnersIncludesUserAndOrganizations(t *testing.T) {
	runner := &fakeRunner{lookup: map[string]internalexec.Result{
		"gh\x00api\x00user":                    {Stdout: `{"login":"jeonghyeon-net"}`},
		"gh\x00api\x00user/orgs\x00--paginate": {Stdout: `[{"login":"creatrip"},{"login":"platform-team"}]`},
	}}
	client := Client{Runner: runner}

	owners, err := client.ListOwners(context.Background())
	if err != nil {
		t.Fatalf("ListOwners returned error: %v", err)
	}

	expected := []string{"jeonghyeon-net", "creatrip", "platform-team"}
	if len(owners) != len(expected) {
		t.Fatalf("expected %d owners, got %d (%#v)", len(expected), len(owners), owners)
	}
	for i := range expected {
		if owners[i] != expected[i] {
			t.Fatalf("owner %d = %q, want %q", i, owners[i], expected[i])
		}
	}
}

func TestClientListRepositoriesForOwnerFiltersArchivedAndSortsByLatestActivity(t *testing.T) {
	runner := &fakeRunner{lookup: map[string]internalexec.Result{
		"gh\x00repo\x00list\x00creatrip\x00--limit\x00200\x00--json\x00nameWithOwner,sshUrl,defaultBranchRef,isArchived,pushedAt": {
			Stdout: `[
				{"nameWithOwner":"creatrip/old-repo","sshUrl":"git@github.com:creatrip/old-repo.git","isArchived":false,"pushedAt":"2024-01-01T00:00:00Z","defaultBranchRef":{"name":"main"}},
				{"nameWithOwner":"creatrip/archived-repo","sshUrl":"git@github.com:creatrip/archived-repo.git","isArchived":true,"pushedAt":"2025-01-01T00:00:00Z","defaultBranchRef":{"name":"main"}},
				{"nameWithOwner":"creatrip/new-repo","sshUrl":"git@github.com:creatrip/new-repo.git","isArchived":false,"pushedAt":"2025-03-01T00:00:00Z","defaultBranchRef":{"name":"main"}}
			]`,
		},
	}}
	client := Client{Runner: runner}

	repos, err := client.ListRepositoriesForOwner(context.Background(), "creatrip")
	if err != nil {
		t.Fatalf("ListRepositoriesForOwner returned error: %v", err)
	}

	expected := []domain.GitHubRepo{
		{NameWithOwner: "creatrip/new-repo", SSHURL: "git@github.com:creatrip/new-repo.git", DefaultBranch: "main"},
		{NameWithOwner: "creatrip/old-repo", SSHURL: "git@github.com:creatrip/old-repo.git", DefaultBranch: "main"},
	}
	if len(repos) != len(expected) {
		t.Fatalf("expected %d repos, got %d (%#v)", len(expected), len(repos), repos)
	}
	for i := range expected {
		if repos[i] != expected[i] {
			t.Fatalf("repo %d = %#v, want %#v", i, repos[i], expected[i])
		}
	}
}

func TestClientListRepositoriesIncludesUserAndOrganizations(t *testing.T) {
	runner := &fakeRunner{lookup: map[string]internalexec.Result{
		"gh\x00api\x00user":                    {Stdout: `{"login":"jeonghyeon-net"}`},
		"gh\x00api\x00user/orgs\x00--paginate": {Stdout: `[{"login":"creatrip"},{"login":"platform-team"}]`},
		"gh\x00repo\x00list\x00jeonghyeon-net\x00--limit\x00200\x00--json\x00nameWithOwner,sshUrl,defaultBranchRef,isArchived,pushedAt": {Stdout: `[{"nameWithOwner":"jeonghyeon-net/baker","sshUrl":"git@github.com:jeonghyeon-net/baker.git","isArchived":false,"pushedAt":"2025-03-01T00:00:00Z","defaultBranchRef":{"name":"main"}}]`},
		"gh\x00repo\x00list\x00creatrip\x00--limit\x00200\x00--json\x00nameWithOwner,sshUrl,defaultBranchRef,isArchived,pushedAt":       {Stdout: `[{"nameWithOwner":"creatrip/admin","sshUrl":"git@github.com:creatrip/admin.git","isArchived":false,"pushedAt":"2025-02-01T00:00:00Z","defaultBranchRef":{"name":"main"}}]`},
		"gh\x00repo\x00list\x00platform-team\x00--limit\x00200\x00--json\x00nameWithOwner,sshUrl,defaultBranchRef,isArchived,pushedAt":  {Stdout: `[]`},
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

	seen := make(map[string]struct{}, len(runner.calls))
	for _, call := range runner.calls {
		seen[call.name+"\x00"+strings.Join(call.args, "\x00")] = struct{}{}
	}
	for _, repo := range expected {
		if _, ok := seen["gh\x00repo\x00list\x00"+strings.Split(repo.NameWithOwner, "/")[0]+"\x00--limit\x00200\x00--json\x00nameWithOwner,sshUrl,defaultBranchRef,isArchived,pushedAt"]; !ok {
			t.Fatalf("missing repo list call for %s", repo.NameWithOwner)
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

	expectedArgs := []string{"repo", "list", "jeonghyeon-net", "--limit", "500", "--json", "nameWithOwner,sshUrl,defaultBranchRef,isArchived,pushedAt"}
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

func TestClientListMyPullRequestsForRepositoryFiltersCrossRepositoryAndSortsByUpdatedAt(t *testing.T) {
	runner := &fakeRunner{lookup: map[string]internalexec.Result{
		"gh\x00pr\x00list\x00--repo\x00creatrip/admin\x00--author\x00@me\x00--state\x00open\x00--json\x00number,title,headRefName,updatedAt,isDraft,isCrossRepository": {
			Stdout: `[
				{"number":11,"title":"older pr","headRefName":"feature/older","updatedAt":"2024-01-01T00:00:00Z","isDraft":false,"isCrossRepository":false},
				{"number":12,"title":"fork pr","headRefName":"feature/fork","updatedAt":"2025-04-01T00:00:00Z","isDraft":false,"isCrossRepository":true},
				{"number":13,"title":"new pr","headRefName":"feature/new","updatedAt":"2025-05-01T00:00:00Z","isDraft":true,"isCrossRepository":false}
			]`,
		},
	}}
	client := Client{Runner: runner}

	prs, err := client.ListMyPullRequestsForRepository(context.Background(), "creatrip", "admin")
	if err != nil {
		t.Fatalf("ListMyPullRequestsForRepository returned error: %v", err)
	}

	expected := []domain.GitHubPullRequest{
		{Number: 13, Title: "new pr", HeadRefName: "feature/new", UpdatedAt: "2025-05-01T00:00:00Z", IsDraft: true},
		{Number: 11, Title: "older pr", HeadRefName: "feature/older", UpdatedAt: "2024-01-01T00:00:00Z", IsDraft: false},
	}
	if len(prs) != len(expected) {
		t.Fatalf("expected %d prs, got %d (%#v)", len(expected), len(prs), prs)
	}
	for i := range expected {
		if prs[i] != expected[i] {
			t.Fatalf("pr %d = %#v, want %#v", i, prs[i], expected[i])
		}
	}
}
