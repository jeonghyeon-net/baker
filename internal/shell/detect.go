package shell

import (
	"fmt"
	"path/filepath"
)

func Detect(shellPath, home string) (string, string, error) {
	switch filepath.Base(shellPath) {
	case "zsh":
		return "zsh", filepath.Join(home, ".zshrc"), nil
	case "bash":
		return "bash", filepath.Join(home, ".bashrc"), nil
	default:
		return "", "", fmt.Errorf("unsupported shell: %s", shellPath)
	}
}
