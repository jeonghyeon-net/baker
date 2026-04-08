package config

import (
	"os"
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

func TestSaveRegistrySyncsTempFileAndParentDirectory(t *testing.T) {
	file := filepath.Join(t.TempDir(), "nested", "workspaces.json")
	registry := Registry{}

	originalFileSync := syncTempFile
	originalDirSync := syncParentDirectory
	t.Cleanup(func() {
		syncTempFile = originalFileSync
		syncParentDirectory = originalDirSync
	})

	fileSyncCalled := false
	dirSyncCalled := false

	syncTempFile = func(f *os.File) error {
		fileSyncCalled = true
		return nil
	}
	syncParentDirectory = func(dir string) error {
		dirSyncCalled = true
		if dir != filepath.Dir(file) {
			t.Fatalf("syncParentDirectory() dir = %q, want %q", dir, filepath.Dir(file))
		}
		return nil
	}

	if err := SaveRegistry(file, registry); err != nil {
		t.Fatalf("SaveRegistry() error = %v", err)
	}

	if !fileSyncCalled {
		t.Fatal("SaveRegistry() did not sync temp file")
	}
	if !dirSyncCalled {
		t.Fatal("SaveRegistry() did not sync parent directory")
	}
}
