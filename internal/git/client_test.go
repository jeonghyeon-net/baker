package git

import (
	"context"
	"errors"
	"testing"

	internalexec "github.com/jeonghyeon-net/baker/internal/exec"
)

type fakeRunnerCall struct {
	name string
	args []string
}

type fakeRunnerResult struct {
	result internalexec.Result
	err    error
}

type fakeRunner struct {
	calls   []fakeRunnerCall
	results []fakeRunnerResult
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) (internalexec.Result, error) {
	f.calls = append(f.calls, fakeRunnerCall{name: name, args: append([]string(nil), args...)})
	if len(f.results) == 0 {
		return internalexec.Result{}, nil
	}
	result := f.results[0]
	f.results = f.results[1:]
	return result.result, result.err
}

func TestCloneBareUsesFilteredCloneAndConfiguresFetchSpec(t *testing.T) {
	runner := &fakeRunner{results: []fakeRunnerResult{{}, {}}}
	client := Client{Runner: runner}

	if err := client.CloneBare(context.Background(), "git@github.com:org/repo.git", "/tmp/repo.git"); err != nil {
		t.Fatalf("CloneBare() error = %v", err)
	}

	if len(runner.calls) != 2 {
		t.Fatalf("call count = %d, want 2", len(runner.calls))
	}
	assertCallArgs(t, runner.calls[0], "git", []string{"clone", "--bare", "--filter=blob:none", "--no-tags", "git@github.com:org/repo.git", "/tmp/repo.git"})
	assertCallArgs(t, runner.calls[1], "git", []string{"--git-dir", "/tmp/repo.git", "config", "remote.origin.fetch", remoteTrackingRefSpec})
}

func TestCloneBareFallsBackWhenFilterUnsupported(t *testing.T) {
	runner := &fakeRunner{results: []fakeRunnerResult{
		{result: internalexec.Result{Stderr: "fatal: server does not support --filter"}, err: errors.New("clone failed")},
		{},
		{},
	}}
	client := Client{Runner: runner}

	if err := client.CloneBare(context.Background(), "git@github.com:org/repo.git", "/tmp/repo.git"); err != nil {
		t.Fatalf("CloneBare() error = %v", err)
	}

	if len(runner.calls) != 3 {
		t.Fatalf("call count = %d, want 3", len(runner.calls))
	}
	assertCallArgs(t, runner.calls[0], "git", []string{"clone", "--bare", "--filter=blob:none", "--no-tags", "git@github.com:org/repo.git", "/tmp/repo.git"})
	assertCallArgs(t, runner.calls[1], "git", []string{"clone", "--bare", "--no-tags", "git@github.com:org/repo.git", "/tmp/repo.git"})
}

func TestFetchAllUsesFilteredFetch(t *testing.T) {
	runner := &fakeRunner{results: []fakeRunnerResult{
		{result: internalexec.Result{Stdout: "origin\n"}},
		{},
	}}
	client := Client{Runner: runner}

	if err := client.FetchAll(context.Background(), "/tmp/repo.git"); err != nil {
		t.Fatalf("FetchAll() error = %v", err)
	}

	if len(runner.calls) != 2 {
		t.Fatalf("call count = %d, want 2", len(runner.calls))
	}
	assertCallArgs(t, runner.calls[1], "git", []string{"--git-dir", "/tmp/repo.git", "fetch", "--prune", "--force", "--filter=blob:none", "origin"})
}

func TestFetchAllFallsBackWhenFilterUnsupported(t *testing.T) {
	runner := &fakeRunner{results: []fakeRunnerResult{
		{result: internalexec.Result{Stdout: "origin\n"}},
		{result: internalexec.Result{Stderr: "fatal: filtering not recognized by server"}, err: errors.New("fetch failed")},
		{},
	}}
	client := Client{Runner: runner}

	if err := client.FetchAll(context.Background(), "/tmp/repo.git"); err != nil {
		t.Fatalf("FetchAll() error = %v", err)
	}

	if len(runner.calls) != 3 {
		t.Fatalf("call count = %d, want 3", len(runner.calls))
	}
	assertCallArgs(t, runner.calls[1], "git", []string{"--git-dir", "/tmp/repo.git", "fetch", "--prune", "--force", "--filter=blob:none", "origin"})
	assertCallArgs(t, runner.calls[2], "git", []string{"--git-dir", "/tmp/repo.git", "fetch", "--prune", "--force", "origin"})
}

func TestSetBranchUpstreamUsesRemoteTrackingRef(t *testing.T) {
	runner := &fakeRunner{results: []fakeRunnerResult{{}}}
	client := Client{Runner: runner}

	if err := client.SetBranchUpstream(context.Background(), "/tmp/worktree", "feature/login", "origin"); err != nil {
		t.Fatalf("SetBranchUpstream() error = %v", err)
	}

	if len(runner.calls) != 1 {
		t.Fatalf("call count = %d, want 1", len(runner.calls))
	}
	assertCallArgs(t, runner.calls[0], "git", []string{"-C", "/tmp/worktree", "branch", "--set-upstream-to", "origin/feature/login", "feature/login"})
}

func TestAddNewBranchWorktreeUsesRemoteTrackingBaseBranch(t *testing.T) {
	runner := &fakeRunner{results: []fakeRunnerResult{{}}}
	client := Client{Runner: runner}

	if err := client.AddNewBranchWorktree(context.Background(), "/tmp/repo.git", "main", "feature/login", "/tmp/worktree"); err != nil {
		t.Fatalf("AddNewBranchWorktree() error = %v", err)
	}

	if len(runner.calls) != 1 {
		t.Fatalf("call count = %d, want 1", len(runner.calls))
	}
	assertCallArgs(t, runner.calls[0], "git", []string{"--git-dir", "/tmp/repo.git", "worktree", "add", "-b", "feature/login", "/tmp/worktree", "origin/main"})
}

func TestDeleteRemoteBranchIgnoresMissingRemoteRef(t *testing.T) {
	runner := &fakeRunner{results: []fakeRunnerResult{{
		result: internalexec.Result{Stderr: "error: unable to delete 'fix/code-to-id': remote ref does not exist\nerror: failed to push some refs to 'github.com:creatrip/product.git'"},
		err:    errors.New("exit status 1"),
	}}}
	client := Client{Runner: runner}

	if err := client.DeleteRemoteBranch(context.Background(), "/tmp/repo.git", "fix/code-to-id"); err != nil {
		t.Fatalf("DeleteRemoteBranch() error = %v, want nil", err)
	}

	if len(runner.calls) != 1 {
		t.Fatalf("call count = %d, want 1", len(runner.calls))
	}
	assertCallArgs(t, runner.calls[0], "git", []string{"--git-dir", "/tmp/repo.git", "push", "origin", "--delete", "fix/code-to-id"})
}

func assertCallArgs(t *testing.T, call fakeRunnerCall, expectedName string, expectedArgs []string) {
	t.Helper()
	if call.name != expectedName {
		t.Fatalf("call name = %q, want %q", call.name, expectedName)
	}
	if len(call.args) != len(expectedArgs) {
		t.Fatalf("args = %v, want %v", call.args, expectedArgs)
	}
	for i, arg := range expectedArgs {
		if call.args[i] != arg {
			t.Fatalf("arg %d = %q, want %q", i, call.args[i], arg)
		}
	}
}
