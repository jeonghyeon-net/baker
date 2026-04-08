package shell

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	hookStart = "# >>> baker initialize >>>"
	hookEnd   = "# <<< baker initialize <<<"
)

func InstallHook(rcPath, shellName string) error {
	if shellName != "zsh" && shellName != "bash" {
		return fmt.Errorf("unsupported shell: %s", shellName)
	}
	data, err := os.ReadFile(rcPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	block := renderHook()
	content := replaceManagedBlock(string(data), block)
	return os.WriteFile(rcPath, []byte(content), 0o644)
}

func renderHook() string {
	return strings.Join([]string{
		hookStart,
		"baker() {",
		"  local baker_result_file baker_status baker_target",
		"  baker_result_file=$(mktemp)",
		"  command baker __shell --result-file \"$baker_result_file\" \"$@\"",
		"  baker_status=$?",
		"  if [ $baker_status -eq 0 ] && [ -s \"$baker_result_file\" ]; then",
		"    baker_target=$(cat \"$baker_result_file\")",
		"    if [ -n \"$baker_target\" ]; then",
		"      cd \"$baker_target\"",
		"      baker_status=$?",
		"    fi",
		"  elif [ $baker_status -eq 0 ] && [ ! -d \"$PWD\" ]; then",
		"    cd \"$HOME\"",
		"    baker_status=$?",
		"  fi",
		"  rm -f \"$baker_result_file\"",
		"  return $baker_status",
		"}",
		hookEnd,
		"",
	}, "\n")
}

func replaceManagedBlock(content, block string) string {
	start := strings.Index(content, hookStart)
	if start < 0 {
		return appendBlock(content, block)
	}
	end := strings.Index(content[start:], hookEnd)
	if end < 0 {
		return content[:start] + block
	}
	end += start + len(hookEnd)
	if end < len(content) && content[end] == '\n' {
		end++
	}
	return content[:start] + block + strings.TrimLeft(content[end:], "\n")
}

func appendBlock(content, block string) string {
	content = strings.TrimRight(content, "\n")
	if content != "" {
		content += "\n\n"
	}
	return content + block
}
