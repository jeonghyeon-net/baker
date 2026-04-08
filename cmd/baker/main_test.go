package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jeonghyeon-net/baker/internal/domain"
)

func TestEnsureProcessOutsidePathMovesToHomeWhenInsideTarget(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	target := filepath.Join(t.TempDir(), "worktree")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	if err := os.Chdir(target); err != nil {
		t.Fatalf("Chdir(target) error = %v", err)
	}
	fallbackPath, err := ensureProcessOutsidePath(target)
	if err != nil {
		t.Fatalf("ensureProcessOutsidePath() error = %v", err)
	}
	if canonicalPath(fallbackPath) != canonicalPath(home) {
		t.Fatalf("fallbackPath = %q, want %q", fallbackPath, home)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() after ensureProcessOutsidePath error = %v", err)
	}
	if canonicalPath(wd) != canonicalPath(home) {
		t.Fatalf("working directory = %q, want %q", wd, home)
	}
}

func TestEnsureProcessOutsidePathDoesNothingWhenOutsideTarget(t *testing.T) {
	outside := t.TempDir()
	target := filepath.Join(t.TempDir(), "worktree")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	if err := os.Chdir(outside); err != nil {
		t.Fatalf("Chdir(outside) error = %v", err)
	}
	fallbackPath, err := ensureProcessOutsidePath(target)
	if err != nil {
		t.Fatalf("ensureProcessOutsidePath() error = %v", err)
	}
	if fallbackPath != "" {
		t.Fatalf("fallbackPath = %q, want empty", fallbackPath)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() after ensureProcessOutsidePath error = %v", err)
	}
	if canonicalPath(wd) != canonicalPath(outside) {
		t.Fatalf("working directory = %q, want %q", wd, outside)
	}
}

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
