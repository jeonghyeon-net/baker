package github

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/jeonghyeon-net/baker/internal/domain"
	internalexec "github.com/jeonghyeon-net/baker/internal/exec"
)

type Runner interface {
	Run(ctx context.Context, name string, args ...string) (internalexec.Result, error)
}

const DefaultRepositoryListLimit = 200

type Client struct {
	Runner              Runner
	RepositoryListLimit int
}

type defaultRunner struct{}

func (defaultRunner) Run(ctx context.Context, name string, args ...string) (internalexec.Result, error) {
	return internalexec.CommandRunner{}.Run(ctx, name, args...)
}

func (c Client) ListRepositories(ctx context.Context) ([]domain.GitHubRepo, error) {
	result, err := c.runner().Run(ctx, "gh", "repo", "list", "--limit", strconv.Itoa(c.repositoryListLimit()), "--json", "nameWithOwner,sshUrl,defaultBranchRef")
	if err != nil {
		return nil, err
	}

	var response []struct {
		NameWithOwner    string `json:"nameWithOwner"`
		SSHURL           string `json:"sshUrl"`
		DefaultBranchRef struct {
			Name string `json:"name"`
		} `json:"defaultBranchRef"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return nil, err
	}

	repos := make([]domain.GitHubRepo, 0, len(response))
	for _, repo := range response {
		repos = append(repos, domain.GitHubRepo{
			NameWithOwner: repo.NameWithOwner,
			SSHURL:        repo.SSHURL,
			DefaultBranch: repo.DefaultBranchRef.Name,
		})
	}

	return repos, nil
}

func (c Client) runner() Runner {
	if c.Runner != nil {
		return c.Runner
	}

	return defaultRunner{}
}

func (c Client) repositoryListLimit() int {
	if c.RepositoryListLimit > 0 {
		return c.RepositoryListLimit
	}

	return DefaultRepositoryListLimit
}
