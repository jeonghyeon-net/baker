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
	localRefs, err := c.runner().Run(ctx, "git", "--git-dir", repoPath, "for-each-ref", "--format=%(refname:short)\tlocal\torigin", "refs/heads")
	if err != nil {
		return nil, err
	}

	remotes, err := c.runner().Run(ctx, "git", "--git-dir", repoPath, "remote")
	if err != nil {
		return nil, err
	}

	outputs := []string{localRefs.Stdout}
	for _, remote := range strings.Split(strings.TrimSpace(remotes.Stdout), "\n") {
		remote = strings.TrimSpace(remote)
		if remote == "" {
			continue
		}

		remoteRefs, err := c.runner().Run(
			ctx,
			"git",
			"--git-dir",
			repoPath,
			"for-each-ref",
			"--exclude=refs/remotes/"+remote+"/HEAD",
			"--format=%(refname:lstrip=3)\tremote\t"+remote,
			"refs/remotes/"+remote,
		)
		if err != nil {
			return nil, err
		}

		outputs = append(outputs, remoteRefs.Stdout)
	}

	return ParseBranches(strings.Join(outputs, ""))
}

func (c Client) runner() Runner {
	if c.Runner != nil {
		return c.Runner
	}

	return defaultRunner{}
}
