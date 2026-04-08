package config

import "testing"

func TestDefaultPaths(t *testing.T) {
	home := "/tmp/example-home"

	got := DefaultPaths(home)

	if got.PiRoot != "/tmp/example-home/.pi" {
		t.Fatalf("PiRoot = %q, want %q", got.PiRoot, "/tmp/example-home/.pi")
	}
	if got.RepositoriesRoot != "/tmp/example-home/.pi/repositories" {
		t.Fatalf("RepositoriesRoot = %q, want %q", got.RepositoriesRoot, "/tmp/example-home/.pi/repositories")
	}
	if got.WorktreesRoot != "/tmp/example-home/.pi/worktrees" {
		t.Fatalf("WorktreesRoot = %q, want %q", got.WorktreesRoot, "/tmp/example-home/.pi/worktrees")
	}
	if got.BakerRoot != "/tmp/example-home/.pi/baker" {
		t.Fatalf("BakerRoot = %q, want %q", got.BakerRoot, "/tmp/example-home/.pi/baker")
	}
	if got.RegistryFile != "/tmp/example-home/.pi/baker/workspaces.json" {
		t.Fatalf("RegistryFile = %q, want %q", got.RegistryFile, "/tmp/example-home/.pi/baker/workspaces.json")
	}
}
