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

func (c Client) CloneBare(ctx context.Context, remoteURL, repoPath string) error {
	_, err := c.runner().Run(ctx, "git", "clone", "--bare", remoteURL, repoPath)
	return err
}

func (c Client) FetchAll(ctx context.Context, repoPath string) error {
	_, err := c.runner().Run(ctx, "git", "--git-dir", repoPath, "fetch", "--all", "--prune")
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

	refs, err := c.runner().Run(ctx, "git", "--git-dir", repoPath, "for-each-ref", "--format=%(refname:short)\tremote\t"+remoteName, "refs/heads")
	if err != nil {
		return nil, err
	}

	return ParseBranches(refs.Stdout)
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
