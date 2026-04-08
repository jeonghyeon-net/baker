package app

import (
	"context"
	"errors"
	"testing"
)

type fakeLookup func(string) (string, error)

func (f fakeLookup) Lookup(name string) (string, error) {
	return f(name)
}

func TestCheckCoreToolsFailsWithoutGit(t *testing.T) {
	checker := Preflight{Lookup: fakeLookup(func(name string) (string, error) {
		if name != "git" {
			t.Fatalf("Lookup() name = %q, want %q", name, "git")
		}
		return "", errors.New("missing")
	})}

	err := checker.CheckCoreTools(context.Background())
	if err == nil || err.Error() != "git is required" {
		t.Fatalf("CheckCoreTools() error = %v, want %q", err, "git is required")
	}
}

func TestCheckGitHubToolsFailsWithoutGH(t *testing.T) {
	checker := Preflight{Lookup: fakeLookup(func(name string) (string, error) {
		if name != "gh" {
			t.Fatalf("Lookup() name = %q, want %q", name, "gh")
		}
		return "", errors.New("missing")
	})}

	err := checker.CheckGitHubTools(context.Background())
	if err == nil || err.Error() != "gh is required for GitHub repository picker" {
		t.Fatalf("CheckGitHubTools() error = %v, want %q", err, "gh is required for GitHub repository picker")
	}
}
