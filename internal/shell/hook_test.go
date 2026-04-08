package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		shellPath string
		home      string
		wantName  string
		wantRC    string
	}{
		{shellPath: "/bin/zsh", home: "/Users/me", wantName: "zsh", wantRC: "/Users/me/.zshrc"},
		{shellPath: "/bin/bash", home: "/Users/me", wantName: "bash", wantRC: "/Users/me/.bashrc"},
	}

	for _, tt := range tests {
		gotName, gotRC, err := Detect(tt.shellPath, tt.home)
		if err != nil {
			t.Fatalf("Detect(%q, %q) error = %v", tt.shellPath, tt.home, err)
		}
		if gotName != tt.wantName || gotRC != tt.wantRC {
			t.Fatalf("Detect(%q, %q) = (%q, %q), want (%q, %q)", tt.shellPath, tt.home, gotName, gotRC, tt.wantName, tt.wantRC)
		}
	}
}

func TestInstallHookIsIdempotent(t *testing.T) {
	rcPath := filepath.Join(t.TempDir(), ".zshrc")

	if err := InstallHook(rcPath, "zsh"); err != nil {
		t.Fatalf("InstallHook() error = %v", err)
	}
	if err := InstallHook(rcPath, "zsh"); err != nil {
		t.Fatalf("InstallHook() second call error = %v", err)
	}

	data, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(data)

	if strings.Count(content, "# >>> baker initialize >>>") != 1 {
		t.Fatalf("hook block count = %d, want 1", strings.Count(content, "# >>> baker initialize >>>"))
	}
	if !strings.Contains(content, "baker()") {
		t.Fatal("hook content missing baker shell function")
	}
	if !strings.Contains(content, "command baker __shell --result-file") {
		t.Fatal("hook content missing __shell invocation")
	}
}
