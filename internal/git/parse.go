package git

import (
	"fmt"
	"strings"

	"github.com/jeonghyeon-net/baker/internal/domain"
)

func ParseBranches(output string) ([]domain.BranchRef, error) {
	var branches []domain.BranchRef

	for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid branch line: %q", line)
		}

		branches = append(branches, domain.BranchRef{
			Name:       parts[0],
			Source:     parts[1],
			RemoteName: parts[2],
		})
	}

	return branches, nil
}

func ParseWorktrees(output string) ([]domain.Worktree, error) {
	normalized := strings.ReplaceAll(output, "\r\n", "\n")
	blocks := strings.Split(normalized, "\n\n")
	worktrees := make([]domain.Worktree, 0, len(blocks))

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		var (
			worktree   domain.Worktree
			isBare     bool
			isDetached bool
		)
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "worktree "):
				worktree.Path = strings.TrimPrefix(line, "worktree ")
			case strings.HasPrefix(line, "HEAD "):
				worktree.HeadSHA = strings.TrimPrefix(line, "HEAD ")
			case strings.HasPrefix(line, "branch refs/heads/"):
				worktree.BranchName = strings.TrimPrefix(line, "branch refs/heads/")
			case line == "bare":
				isBare = true
			case line == "detached":
				isDetached = true
			}
		}

		if worktree.BranchName == "" {
			if isBare {
				continue
			}
			if isDetached {
				worktrees = append(worktrees, worktree)
				continue
			}
			return nil, fmt.Errorf("worktree block missing branch name: %q", block)
		}

		worktrees = append(worktrees, worktree)
	}

	return worktrees, nil
}
