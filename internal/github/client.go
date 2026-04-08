package github

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"time"

	"github.com/jeonghyeon-net/baker/internal/domain"
	internalexec "github.com/jeonghyeon-net/baker/internal/exec"
	"golang.org/x/sync/errgroup"
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

type ghRepository struct {
	NameWithOwner    string `json:"nameWithOwner"`
	SSHURL           string `json:"sshUrl"`
	IsArchived       bool   `json:"isArchived"`
	PushedAt         string `json:"pushedAt"`
	DefaultBranchRef struct {
		Name string `json:"name"`
	} `json:"defaultBranchRef"`
}

type ghPullRequest struct {
	Number            int    `json:"number"`
	Title             string `json:"title"`
	HeadRefName       string `json:"headRefName"`
	UpdatedAt         string `json:"updatedAt"`
	IsDraft           bool   `json:"isDraft"`
	ReviewDecision    string `json:"reviewDecision"`
	IsCrossRepository bool   `json:"isCrossRepository"`
}

func (defaultRunner) Run(ctx context.Context, name string, args ...string) (internalexec.Result, error) {
	return internalexec.CommandRunner{}.Run(ctx, name, args...)
}

func (c Client) ListOwners(ctx context.Context) ([]string, error) {
	return c.listOwners(ctx)
}

func (c Client) ListRepositoriesForOwner(ctx context.Context, owner string) ([]domain.GitHubRepo, error) {
	return c.listRepositoriesForOwner(ctx, owner)
}

func (c Client) ListMyPullRequestsForRepository(ctx context.Context, owner, repo string) ([]domain.GitHubPullRequest, error) {
	result, err := c.runner().Run(ctx, "gh", "pr", "list", "--repo", owner+"/"+repo, "--author", "@me", "--state", "open", "--json", "number,title,headRefName,updatedAt,isDraft,reviewDecision,isCrossRepository")
	if err != nil {
		return nil, err
	}

	var response []ghPullRequest
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return nil, err
	}

	sort.SliceStable(response, func(i, j int) bool {
		return parseTimestamp(response[i].UpdatedAt).After(parseTimestamp(response[j].UpdatedAt))
	})

	prs := make([]domain.GitHubPullRequest, 0, len(response))
	for _, pr := range response {
		if pr.IsCrossRepository {
			continue
		}
		prs = append(prs, domain.GitHubPullRequest{
			Number:            pr.Number,
			Title:             pr.Title,
			HeadRefName:       pr.HeadRefName,
			UpdatedAt:         pr.UpdatedAt,
			IsDraft:           pr.IsDraft,
			ReviewDecision:    pr.ReviewDecision,
			IsCrossRepository: pr.IsCrossRepository,
		})
	}

	return prs, nil
}

func (c Client) ListRepositories(ctx context.Context) ([]domain.GitHubRepo, error) {
	owners, err := c.listOwners(ctx)
	if err != nil {
		return nil, err
	}

	reposByOwner := make([][]domain.GitHubRepo, len(owners))
	group, groupCtx := errgroup.WithContext(ctx)
	for i, owner := range owners {
		i, owner := i, owner
		group.Go(func() error {
			ownerRepos, err := c.listRepositoriesForOwner(groupCtx, owner)
			if err != nil {
				return err
			}
			reposByOwner[i] = ownerRepos
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	var repos []domain.GitHubRepo
	for _, ownerRepos := range reposByOwner {
		for _, repo := range ownerRepos {
			if _, exists := seen[repo.NameWithOwner]; exists {
				continue
			}
			seen[repo.NameWithOwner] = struct{}{}
			repos = append(repos, repo)
		}
	}

	return repos, nil
}

func (c Client) listOwners(ctx context.Context) ([]string, error) {
	userResult, err := c.runner().Run(ctx, "gh", "api", "user")
	if err != nil {
		return nil, err
	}
	var user struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal([]byte(userResult.Stdout), &user); err != nil {
		return nil, err
	}

	orgsResult, err := c.runner().Run(ctx, "gh", "api", "user/orgs", "--paginate")
	if err != nil {
		return nil, err
	}
	var orgs []struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal([]byte(orgsResult.Stdout), &orgs); err != nil {
		return nil, err
	}

	owners := make([]string, 0, 1+len(orgs))
	if user.Login != "" {
		owners = append(owners, user.Login)
	}
	for _, org := range orgs {
		if org.Login != "" {
			owners = append(owners, org.Login)
		}
	}
	return owners, nil
}

func (c Client) listRepositoriesForOwner(ctx context.Context, owner string) ([]domain.GitHubRepo, error) {
	result, err := c.runner().Run(ctx, "gh", "repo", "list", owner, "--limit", strconv.Itoa(c.repositoryListLimit()), "--json", "nameWithOwner,sshUrl,defaultBranchRef,isArchived,pushedAt")
	if err != nil {
		return nil, err
	}

	var response []ghRepository
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return nil, err
	}

	sort.SliceStable(response, func(i, j int) bool {
		return parseTimestamp(response[i].PushedAt).After(parseTimestamp(response[j].PushedAt))
	})

	repos := make([]domain.GitHubRepo, 0, len(response))
	for _, repo := range response {
		if repo.IsArchived {
			continue
		}
		repos = append(repos, domain.GitHubRepo{
			NameWithOwner: repo.NameWithOwner,
			SSHURL:        repo.SSHURL,
			DefaultBranch: repo.DefaultBranchRef.Name,
		})
	}

	return repos, nil
}

func parseTimestamp(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
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
