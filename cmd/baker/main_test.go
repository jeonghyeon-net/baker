package main

import (
	"testing"

	"github.com/jeonghyeon-net/baker/internal/domain"
)

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
