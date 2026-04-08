package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetect(t *testing.T) {
	tests := []struct{ shellPath, home, wantName, wantRC string }{
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
	for range 2 {
		if err := InstallHook(rcPath, "zsh"); err != nil {
			t.Fatalf("InstallHook() error = %v", err)
		}
	}
	content := readHookFile(t, rcPath)
	if strings.Count(content, hookStart) != 1 || strings.Count(content, hookEnd) != 1 {
		t.Fatalf("managed block markers = (%d, %d), want (1, 1)", strings.Count(content, hookStart), strings.Count(content, hookEnd))
	}
	assertHookContent(t, content)
}

func TestInstallHookRepairsPartialManagedBlock(t *testing.T) {
	rcPath := filepath.Join(t.TempDir(), ".bashrc")
	if err := os.WriteFile(rcPath, []byte("export PATH=/tmp/bin\n\n"+hookStart+"\npartial\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := InstallHook(rcPath, "bash"); err != nil {
		t.Fatalf("InstallHook() error = %v", err)
	}
	content := readHookFile(t, rcPath)
	if strings.Count(content, hookStart) != 1 || strings.Count(content, hookEnd) != 1 {
		t.Fatalf("managed block markers = (%d, %d), want (1, 1)", strings.Count(content, hookStart), strings.Count(content, hookEnd))
	}
	assertHookContent(t, content)
}

func readHookFile(t *testing.T, rcPath string) string {
	t.Helper()
	data, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	return string(data)
}

func assertHookContent(t *testing.T, content string) {
	t.Helper()
	for _, want := range []string{"baker()", "local baker_result_file baker_status baker_target", "command baker __shell --result-file \"$baker_result_file\" \"$@\"", "elif [ $baker_status -eq 0 ] && [ ! -d \"$PWD\" ]; then", "cd \"$HOME\""} {
		if !strings.Contains(content, want) {
			t.Fatalf("hook content missing %q", want)
		}
	}
	if strings.Contains(content, "|| true") {
		t.Fatal("hook content should propagate cd failures")
	}
}
