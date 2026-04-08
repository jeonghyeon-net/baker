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
	"github.com/jeonghyeon-net/baker/internal/domain"
	bakergit "github.com/jeonghyeon-net/baker/internal/git"
	bakergithub "github.com/jeonghyeon-net/baker/internal/github"
	bakershell "github.com/jeonghyeon-net/baker/internal/shell"
	"github.com/jeonghyeon-net/baker/internal/ui"
	bakerworkspace "github.com/jeonghyeon-net/baker/internal/workspace"
	bakerworktree "github.com/jeonghyeon-net/baker/internal/worktree"
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

		selectedPath, err := runShellMode(context.Background())
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
		selectedPath, err := runShellMode(context.Background())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if selectedPath != "" {
			fmt.Println(selectedPath)
		}
	}
}

func runShellMode(ctx context.Context) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	paths := config.DefaultPaths(home)
	registry, err := config.LoadRegistry(paths.RegistryFile)
	if err != nil {
		return "", err
	}

	worktrees, err := loadWorktreeItems(paths, registry)
	if err != nil {
		return "", err
	}

	selectedPath, action, err := runWorktreeSelection(worktrees)
	if err != nil {
		return "", err
	}
	if selectedPath != "" {
		return selectedPath, nil
	}
	if action == "create-workspace-github" {
		return createInitialWorktree(ctx, paths, registry)
	}
	return "", nil
}

func runWorktreeSelection(worktrees []ui.WorktreeItem) (string, string, error) {
	finalModel, err := tea.NewProgram(ui.NewModel(ui.State{Screen: ui.ScreenWorktrees, Worktrees: worktrees})).Run()
	if err != nil {
		return "", "", err
	}

	selected, ok := finalModel.(ui.Model)
	if !ok {
		return "", "", fmt.Errorf("unexpected ui model type %T", finalModel)
	}
	return selected.SelectedPath, selected.SelectedAction, nil
}

func createInitialWorktree(ctx context.Context, paths config.Paths, registry config.Registry) (string, error) {
	githubClient := bakergithub.Client{}
	repos, err := githubClient.ListRepositories(ctx)
	if err != nil {
		return "", err
	}
	if len(repos) == 0 {
		return "", nil
	}

	selectedRepo, err := runRepositorySelection(repos)
	if err != nil || selectedRepo == nil {
		return "", err
	}

	gitClient := bakergit.Client{}
	workspaceService := bakerworkspace.Service{Git: gitClient, Paths: paths}
	worktreeService := bakerworktree.Service{Git: gitClient, Paths: paths}

	workspace, updatedRegistry, err := ensureWorkspace(ctx, paths, registry, workspaceService, *selectedRepo)
	if err != nil {
		return "", err
	}
	if err := config.SaveRegistry(paths.RegistryFile, updatedRegistry); err != nil {
		return "", err
	}
	if err := workspaceService.Sync(ctx, workspace); err != nil {
		return "", err
	}

	branches, err := gitClient.ListBranches(ctx, workspace.RepositoryPath)
	if err != nil {
		return "", err
	}
	branchName := defaultBranchName(*selectedRepo, workspace, branches)
	if branchName == "" {
		return "", fmt.Errorf("no branches available for %s", selectedRepo.NameWithOwner)
	}

	result, err := worktreeService.CreateFromExistingBranch(ctx, workspace, branches, branchName, worktreeNameForBranch(branchName))
	if err != nil {
		return "", err
	}
	return result.Path, nil
}

func runRepositorySelection(repos []domain.GitHubRepo) (*domain.GitHubRepo, error) {
	names := make([]string, 0, len(repos))
	for _, repo := range repos {
		names = append(names, repo.NameWithOwner)
	}

	finalModel, err := tea.NewProgram(ui.NewModel(ui.State{Screen: ui.ScreenWorkspaceGitHubPicker, Repositories: names})).Run()
	if err != nil {
		return nil, err
	}

	selected, ok := finalModel.(ui.Model)
	if !ok {
		return nil, fmt.Errorf("unexpected ui model type %T", finalModel)
	}
	if selected.SelectedPath == "" {
		return nil, nil
	}

	for _, repo := range repos {
		if repo.NameWithOwner == selected.SelectedPath {
			return &repo, nil
		}
	}
	return nil, fmt.Errorf("selected repository not found: %s", selected.SelectedPath)
}

func ensureWorkspace(ctx context.Context, paths config.Paths, registry config.Registry, workspaceService bakerworkspace.Service, repo domain.GitHubRepo) (domain.Workspace, config.Registry, error) {
	suggestedName := strings.ReplaceAll(repo.NameWithOwner, "/", "-")
	for i, workspace := range registry.Workspaces {
		if workspace.RemoteURL == repo.SSHURL || workspace.Owner+"/"+workspace.Repo == repo.NameWithOwner {
			if workspace.DefaultBranch == "" && repo.DefaultBranch != "" {
				registry.Workspaces[i].DefaultBranch = repo.DefaultBranch
				workspace = registry.Workspaces[i]
			}
			return workspace, registry, nil
		}
	}

	workspaceName := uniqueWorkspaceName(registry.Workspaces, suggestedName)
	workspace, err := workspaceService.CreateFromGitHubRepo(ctx, repo, workspaceName)
	if err != nil {
		return domain.Workspace{}, registry, err
	}
	registry.Workspaces = append(registry.Workspaces, workspace)
	return workspace, registry, nil
}

func uniqueWorkspaceName(workspaces []domain.Workspace, suggestedName string) string {
	used := make(map[string]struct{}, len(workspaces))
	for _, workspace := range workspaces {
		used[workspace.Name] = struct{}{}
	}
	if _, exists := used[suggestedName]; !exists {
		return suggestedName
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", suggestedName, i)
		if _, exists := used[candidate]; !exists {
			return candidate
		}
	}
}

func defaultBranchName(repo domain.GitHubRepo, workspace domain.Workspace, branches []domain.BranchRef) string {
	for _, candidate := range []string{repo.DefaultBranch, workspace.DefaultBranch} {
		if candidate != "" {
			for _, branch := range branches {
				if branch.Name == candidate {
					return candidate
				}
			}
		}
	}
	if len(branches) > 0 {
		return branches[0].Name
	}
	return ""
}

func worktreeNameForBranch(branch string) string {
	return strings.NewReplacer("/", "-", `\\`, "-").Replace(branch)
}

func loadWorktreeItems(paths config.Paths, registry config.Registry) ([]ui.WorktreeItem, error) {
	type workspaceItems struct {
		name  string
		items []ui.WorktreeItem
	}

	var groups []workspaceItems
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

		var items []ui.WorktreeItem
		for _, entry := range entries {
			if entry.IsDir() {
				items = append(items, ui.WorktreeItem{
					Label:      "  " + entry.Name(),
					Path:       filepath.Join(workspaceRoot, entry.Name()),
					Selectable: true,
				})
			}
		}
		if len(items) == 0 {
			continue
		}

		sort.Slice(items, func(i, j int) bool { return items[i].Label < items[j].Label })
		groups = append(groups, workspaceItems{name: workspace.Name, items: items})
	}

	sort.Slice(groups, func(i, j int) bool { return groups[i].name < groups[j].name })

	var worktrees []ui.WorktreeItem
	for _, group := range groups {
		worktrees = append(worktrees, ui.WorktreeItem{Label: group.name, Selectable: false})
		worktrees = append(worktrees, group.items...)
	}
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
