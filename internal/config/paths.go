package config

import "path/filepath"

type Paths struct {
	PiRoot           string
	RepositoriesRoot string
	WorktreesRoot    string
	BakerRoot        string
	RegistryFile     string
}

func DefaultPaths(home string) Paths {
	piRoot := filepath.Join(home, ".pi")
	bakerRoot := filepath.Join(piRoot, "baker")

	return Paths{
		PiRoot:           piRoot,
		RepositoriesRoot: filepath.Join(piRoot, "repositories"),
		WorktreesRoot:    filepath.Join(piRoot, "worktrees"),
		BakerRoot:        bakerRoot,
		RegistryFile:     filepath.Join(bakerRoot, "workspaces.json"),
	}
}
