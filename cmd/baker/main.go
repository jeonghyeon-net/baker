package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jeonghyeon-net/baker/internal/app"
	"github.com/jeonghyeon-net/baker/internal/config"
	bakershell "github.com/jeonghyeon-net/baker/internal/shell"
	"github.com/jeonghyeon-net/baker/internal/ui"
)

type bootstrapShell struct{}

func (bootstrapShell) Ensure() (bool, string, error) {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		return false, "", fmt.Errorf("SHELL is not set")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return false, "", err
	}

	shellName, rcPath, err := bakershell.Detect(shellPath, home)
	if err != nil {
		return false, "", err
	}

	before, err := readFileOrEmpty(rcPath)
	if err != nil {
		return false, "", err
	}
	if err := bakershell.InstallHook(rcPath, shellName); err != nil {
		return false, "", err
	}
	after, err := readFileOrEmpty(rcPath)
	if err != nil {
		return false, "", err
	}
	if before != after {
		return false, fmt.Sprintf("source %s", rcPath), nil
	}

	return true, "", nil
}

func readFileOrEmpty(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "__shell" {
		shellFlags := flag.NewFlagSet("__shell", flag.ExitOnError)
		resultFile := shellFlags.String("result-file", "", "path to shell result file")
		_ = shellFlags.Parse(os.Args[2:])

		selectedPath, err := runShellMode()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if *resultFile != "" && selectedPath != "" {
			if err := app.WriteShellResult(*resultFile, selectedPath); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		return
	}

	result, err := app.Application{Shell: bootstrapShell{}}.Run(context.Background(), app.Options{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if result.Message != "" {
		fmt.Println(result.Message)
		return
	}
	if result.Mode == app.ModeInteractive {
		selectedPath, err := runShellMode()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if selectedPath != "" {
			fmt.Println(selectedPath)
		}
	}
}

func runShellMode() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	paths := config.DefaultPaths(home)
	worktrees, err := loadWorktreePaths(paths)
	if err != nil {
		return "", err
	}

	model := ui.NewModel(ui.State{Screen: ui.ScreenWorktrees, Worktrees: worktrees})
	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return "", err
	}

	selected, ok := finalModel.(ui.Model)
	if !ok {
		return "", fmt.Errorf("unexpected ui model type %T", finalModel)
	}
	return selected.SelectedPath, nil
}

func loadWorktreePaths(paths config.Paths) ([]string, error) {
	registry, err := config.LoadRegistry(paths.RegistryFile)
	if err != nil {
		return nil, err
	}

	var worktrees []string
	for _, workspace := range registry.Workspaces {
		workspaceRoot, ok := managedWorkspaceRoot(paths.WorktreesRoot, workspace.Name)
		if !ok {
			continue
		}
		entries, err := os.ReadDir(workspaceRoot)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				worktrees = append(worktrees, filepath.Join(workspaceRoot, entry.Name()))
			}
		}
	}

	sort.Strings(worktrees)
	return worktrees, nil
}

func managedWorkspaceRoot(worktreesRoot, workspaceName string) (string, bool) {
	if workspaceName == "" || workspaceName == "." || workspaceName == ".." || filepath.IsAbs(workspaceName) || strings.ContainsAny(workspaceName, `/\\`) {
		return "", false
	}

	root := filepath.Clean(worktreesRoot)
	workspaceRoot := filepath.Join(root, workspaceName)
	rel, err := filepath.Rel(root, workspaceRoot)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return workspaceRoot, true
}
