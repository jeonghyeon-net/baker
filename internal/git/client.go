package git

import (
	"context"
	"strings"

	"github.com/jeonghyeon-net/baker/internal/domain"
	internalexec "github.com/jeonghyeon-net/baker/internal/exec"
)

type Runner interface {
	Run(ctx context.Context, name string, args ...string) (internalexec.Result, error)
}

type defaultRunner struct{}

func (defaultRunner) Run(ctx context.Context, name string, args ...string) (internalexec.Result, error) {
	return internalexec.CommandRunner{}.Run(ctx, name, args...)
}

type Client struct {
	Runner Runner
}

const (
	remoteTrackingRefSpec = "+refs/heads/*:refs/remotes/origin/*"
	partialCloneFilter    = "blob:none"
)

func (c Client) CloneBare(ctx context.Context, remoteURL, repoPath string) error {
	if err := c.cloneBareOptimized(ctx, remoteURL, repoPath); err != nil {
		return err
	}

	_, err := c.runner().Run(ctx, "git", "--git-dir", repoPath, "config", "remote.origin.fetch", remoteTrackingRefSpec)
	return err
}

func (c Client) cloneBareOptimized(ctx context.Context, remoteURL, repoPath string) error {
	cloneArgs := []string{"clone", "--bare", "--filter=" + partialCloneFilter, "--no-tags", remoteURL, repoPath}
	result, err := c.runner().Run(ctx, "git", cloneArgs...)
	if err == nil {
		return nil
	}
	if !shouldRetryWithoutFilter(result) {
		return err
	}

	_, fallbackErr := c.runner().Run(ctx, "git", "clone", "--bare", "--no-tags", remoteURL, repoPath)
	return fallbackErr
}

func (c Client) FetchAll(ctx context.Context, repoPath string) error {
	remotes, err := c.runner().Run(ctx, "git", "--git-dir", repoPath, "remote")
	if err != nil {
		return err
	}

	remoteName := firstRemoteName(remotes.Stdout)
	if remoteName == "" {
		remoteName = "origin"
	}

	fetchArgs := []string{"--git-dir", repoPath, "fetch", "--prune", "--force", "--filter=" + partialCloneFilter, remoteName}
	result, err := c.runner().Run(ctx, "git", fetchArgs...)
	if err == nil {
		return nil
	}
	if !shouldRetryWithoutFilter(result) {
		return err
	}

	_, fallbackErr := c.runner().Run(ctx, "git", "--git-dir", repoPath, "fetch", "--prune", "--force", remoteName)
	return fallbackErr
}

func (c Client) AddExistingBranchWorktree(ctx context.Context, repoPath, branch, worktreePath string) error {
	_, err := c.runner().Run(ctx, "git", "--git-dir", repoPath, "worktree", "add", worktreePath, branch)
	return err
}

func (c Client) SetBranchUpstream(ctx context.Context, worktreePath, branch, remoteName string) error {
	if remoteName == "" {
		remoteName = "origin"
	}
	_, err := c.runner().Run(ctx, "git", "-C", worktreePath, "branch", "--set-upstream-to", remoteName+"/"+branch, branch)
	return err
}

func (c Client) AddNewBranchWorktree(ctx context.Context, repoPath, baseBranch, newBranch, worktreePath string) error {
	_, err := c.runner().Run(ctx, "git", "--git-dir", repoPath, "worktree", "add", "-b", newBranch, worktreePath, baseBranch)
	return err
}

func (c Client) PushBranch(ctx context.Context, worktreePath, branch string) error {
	_, err := c.runner().Run(ctx, "git", "-C", worktreePath, "push", "-u", "origin", branch)
	return err
}

func (c Client) ListBranches(ctx context.Context, repoPath string) ([]domain.BranchRef, error) {
	remotes, err := c.runner().Run(ctx, "git", "--git-dir", repoPath, "remote")
	if err != nil {
		return nil, err
	}

	remoteName := firstRemoteName(remotes.Stdout)
	if remoteName == "" {
		remoteName = "origin"
	}

	refs, err := c.runner().Run(ctx, "git", "--git-dir", repoPath, "for-each-ref", "--format=%(refname:lstrip=3)\tremote\t"+remoteName, "refs/remotes/"+remoteName)
	if err != nil {
		return nil, err
	}

	branches, err := ParseBranches(refs.Stdout)
	if err != nil {
		return nil, err
	}

	filtered := branches[:0]
	for _, branch := range branches {
		if branch.Name == "HEAD" {
			continue
		}
		filtered = append(filtered, branch)
	}

	return filtered, nil
}

func (c Client) ListWorktrees(ctx context.Context, repoPath string) ([]domain.Worktree, error) {
	result, err := c.runner().Run(ctx, "git", "--git-dir", repoPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	return ParseWorktrees(result.Stdout)
}

func (c Client) RemoveWorktree(ctx context.Context, repoPath, worktreePath string, force bool) error {
	args := []string{"--git-dir", repoPath, "worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, worktreePath)

	_, err := c.runner().Run(ctx, "git", args...)
	return err
}

func (c Client) DeleteLocalBranch(ctx context.Context, repoPath, branch string, force bool) error {
	deleteFlag := "-d"
	if force {
		deleteFlag = "-D"
	}

	_, err := c.runner().Run(ctx, "git", "--git-dir", repoPath, "branch", deleteFlag, branch)
	return err
}

func (c Client) DeleteRemoteBranch(ctx context.Context, repoPath, branch string) error {
	_, err := c.runner().Run(ctx, "git", "--git-dir", repoPath, "push", "origin", "--delete", branch)
	return err
}

func shouldRetryWithoutFilter(result internalexec.Result) bool {
	output := strings.ToLower(result.Stdout + "\n" + result.Stderr)
	for _, needle := range []string{
		"does not support filter",
		"does not support --filter",
		"filtering not recognized by server",
		"unknown option `filter`",
		"unknown option 'filter'",
		"invalid filter-spec",
		"filter-spec",
	} {
		if strings.Contains(output, needle) {
			return true
		}
	}
	return false
}

func firstRemoteName(output string) string {
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}

	return ""
}

func (c Client) runner() Runner {
	if c.Runner != nil {
		return c.Runner
	}

	return defaultRunner{}
}
