package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type fakeShellInstaller struct {
	installed bool
}

func (f *fakeShellInstaller) Ensure() (bool, string, error) {
	f.installed = true
	return false, "source ~/.zshrc", nil
}

func TestRunInstallsHookAndReturnsInstruction(t *testing.T) {
	application := Application{Shell: &fakeShellInstaller{}}

	result, err := application.Run(context.Background(), Options{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Mode != ModeNeedsSource {
		t.Fatalf("Mode = %q", result.Mode)
	}
	if result.Message != "source ~/.zshrc" {
		t.Fatalf("Message = %q", result.Message)
	}
}

func TestWriteShellResultFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "result.txt")
	if err := WriteShellResult(file, "/tmp/.pi/worktrees/repo/main"); err != nil {
		t.Fatalf("WriteShellResult() error = %v", err)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "/tmp/.pi/worktrees/repo/main" {
		t.Fatalf("result file = %q", string(data))
	}
}
