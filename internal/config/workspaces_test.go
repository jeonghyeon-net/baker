package config

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jeonghyeon-net/baker/internal/domain"
)

func TestRegistryRoundTrip(t *testing.T) {
	file := filepath.Join(t.TempDir(), "nested", "workspaces.json")
	want := Registry{
		Workspaces: []domain.Workspace{
			{
				Name:           "baker",
				RemoteURL:      "git@github.com:jeonghyeon-net/baker.git",
				Owner:          "jeonghyeon-net",
				Repo:           "baker",
				DefaultBranch:  "main",
				RepositoryPath: "/tmp/repos/baker",
			},
		},
	}

	if err := SaveRegistry(file, want); err != nil {
		t.Fatalf("SaveRegistry() error = %v", err)
	}

	got, err := LoadRegistry(file)
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadRegistry() = %#v, want %#v", got, want)
	}
}
