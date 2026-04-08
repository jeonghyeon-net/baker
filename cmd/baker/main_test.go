package main

import (
	"testing"

	"github.com/jeonghyeon-net/baker/internal/domain"
)

func TestPrioritizedBranchNames(t *testing.T) {
	tests := []struct {
		name     string
		branches []string
		want     []string
	}{
		{name: "moves development production and main to front", branches: []string{"release", "main", "production", "feature/login", "development"}, want: []string{"development", "production", "main", "release", "feature/login"}},
		{name: "keeps missing priority branches absent", branches: []string{"release", "feature/login"}, want: []string{"release", "feature/login"}},
		{name: "keeps only present priority branches", branches: []string{"release", "main", "feature/login"}, want: []string{"main", "release", "feature/login"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prioritizedBranchNames(tt.branches)
			if len(got) != len(tt.want) {
				t.Fatalf("len(prioritizedBranchNames()) = %d, want %d (%#v)", len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("prioritizedBranchNames()[%d] = %q, want %q (%#v)", i, got[i], tt.want[i], got)
				}
			}
		})
	}
}

func TestPullRequestStatusLabel(t *testing.T) {
	tests := []struct {
		name string
		pr   domain.GitHubPullRequest
		want string
	}{
		{name: "draft", pr: domain.GitHubPullRequest{IsDraft: true}, want: "초안"},
		{name: "approved", pr: domain.GitHubPullRequest{ReviewDecision: "APPROVED"}, want: "승인"},
		{name: "changes requested", pr: domain.GitHubPullRequest{ReviewDecision: "CHANGES_REQUESTED"}, want: "수정 요청"},
		{name: "review required", pr: domain.GitHubPullRequest{ReviewDecision: "REVIEW_REQUIRED"}, want: "리뷰 대기"},
		{name: "unknown", pr: domain.GitHubPullRequest{ReviewDecision: "COMMENTED"}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pullRequestStatusLabel(tt.pr); got != tt.want {
				t.Fatalf("pullRequestStatusLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}
