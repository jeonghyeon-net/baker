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
	content := string(data)
	if strings.Contains(content, hookStart) {
		return nil
	}
	block := renderHook()
	content = strings.TrimRight(content, "\n")
	if content != "" {
		content += "\n\n"
	}
	content += block
	return os.WriteFile(rcPath, []byte(content), 0o644)
}

func renderHook() string {
	return strings.Join([]string{
		hookStart,
		"baker() {",
		"  baker_result_file=$(mktemp)",
		"  command baker __shell --result-file \"$baker_result_file\" \"$@\"",
		"  baker_status=$?",
		"  if [ $baker_status -eq 0 ] && [ -s \"$baker_result_file\" ]; then",
		"    baker_target=$(cat \"$baker_result_file\")",
		"    if [ -n \"$baker_target\" ]; then",
		"      cd \"$baker_target\" || true",
		"    fi",
		"  fi",
		"  rm -f \"$baker_result_file\"",
		"  return $baker_status",
		"}",
		hookEnd,
		"",
	}, "\n")
}
