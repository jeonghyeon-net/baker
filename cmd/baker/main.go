package main

import (
	"bufio"
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
	gitClient := bakergit.Client{}
	workspaceService := bakerworkspace.Service{Git: gitClient, Paths: paths}
	worktreeService := bakerworktree.Service{Git: gitClient, Paths: paths}
	githubClient := bakergithub.Client{}

	for {
		registry, err := config.LoadRegistry(paths.RegistryFile)
		if err != nil {
			return "", err
		}

		worktrees, err := loadWorktreeItems(ctx, paths, registry, gitClient)
		if err != nil {
			return "", err
		}

		selection, err := runWorktreeSelection(worktrees)
		if err != nil {
			return "", err
		}
		if selection.SelectedPath != "" && selection.SelectedAction == "" {
			return selection.SelectedPath, nil
		}

		switch selection.SelectedAction {
		case "add-workspace":
			if err := addWorkspace(ctx, paths, registry, githubClient, workspaceService); err != nil {
				return "", err
			}
			continue
		case "create-worktree":
			path, err := createWorktreeForWorkspace(ctx, paths, registry, selection.SelectedWorkspace, gitClient, workspaceService, worktreeService)
			if err != nil {
				return "", err
			}
			if path != "" {
				return path, nil
			}
			continue
		case "delete-worktree":
			if err := deleteSelectedWorktree(ctx, paths, registry, selection, worktreeService); err != nil {
				return "", err
			}
			continue
		default:
			return "", nil
		}
	}
}

func runWorktreeSelection(worktrees []ui.WorktreeItem) (ui.Model, error) {
	finalModel, err := tea.NewProgram(ui.NewModel(ui.State{Screen: ui.ScreenWorktrees, Worktrees: worktrees})).Run()
	if err != nil {
		return ui.Model{}, err
	}

	selected, ok := finalModel.(ui.Model)
	if !ok {
		return ui.Model{}, fmt.Errorf("unexpected ui model type %T", finalModel)
	}
	return selected, nil
}

func addWorkspace(ctx context.Context, paths config.Paths, registry config.Registry, githubClient bakergithub.Client, workspaceService bakerworkspace.Service) error {
	repos, err := githubClient.ListRepositories(ctx)
	if err != nil {
		return err
	}
	if len(repos) == 0 {
		fmt.Println("표시할 GitHub 저장소가 없습니다.")
		return nil
	}

	selectedRepo, err := runRepositorySelection(repos)
	if err != nil || selectedRepo == nil {
		return err
	}

	_, updatedRegistry, err := ensureWorkspace(ctx, paths, registry, workspaceService, *selectedRepo)
	if err != nil {
		return err
	}
	if err := config.SaveRegistry(paths.RegistryFile, updatedRegistry); err != nil {
		return err
	}
	fmt.Printf("workspace added: %s\n", strings.ReplaceAll(selectedRepo.NameWithOwner, "/", "-"))
	return nil
}

func createWorktreeForWorkspace(ctx context.Context, paths config.Paths, registry config.Registry, workspaceName string, gitClient bakergit.Client, workspaceService bakerworkspace.Service, worktreeService bakerworktree.Service) (string, error) {
	workspace, ok := findWorkspace(registry, workspaceName)
	if !ok {
		return "", fmt.Errorf("workspace not found: %s", workspaceName)
	}
	if err := workspaceService.Sync(ctx, workspace); err != nil {
		return "", err
	}
	branches, err := gitClient.ListBranches(ctx, workspace.RepositoryPath)
	if err != nil {
		return "", err
	}
	if len(branches) == 0 {
		return "", fmt.Errorf("no branches available for workspace %s", workspaceName)
	}

	mode, err := promptCreateMode()
	if err != nil {
		return "", err
	}
	if mode == "" {
		return "", nil
	}

	branchNames := make([]string, 0, len(branches))
	for _, branch := range branches {
		branchNames = append(branchNames, branch.Name)
	}

	if mode == "existing" {
		selectedBranch, err := runBranchSelection(branchNames, false)
		if err != nil || selectedBranch == "" {
			return "", err
		}
		result, err := worktreeService.CreateFromExistingBranch(ctx, workspace, branches, selectedBranch, worktreeNameForBranch(selectedBranch))
		if err != nil {
			return "", err
		}
		return result.Path, nil
	}

	baseBranch, err := runBranchSelection(branchNames, false)
	if err != nil || baseBranch == "" {
		return "", err
	}
	newBranch, err := promptText("new branch name")
	if err != nil {
		return "", err
	}
	if newBranch == "" {
		return "", nil
	}
	result, err := worktreeService.CreateFromNewBranch(ctx, workspace, baseBranch, newBranch, worktreeNameForBranch(newBranch))
	if err != nil {
		return result.Path, err
	}
	return result.Path, nil
}

func deleteSelectedWorktree(ctx context.Context, paths config.Paths, registry config.Registry, selection ui.Model, worktreeService bakerworktree.Service) error {
	workspace, ok := findWorkspace(registry, selection.SelectedWorkspace)
	if !ok {
		return fmt.Errorf("workspace not found: %s", selection.SelectedWorkspace)
	}

	mode, err := runDeleteModeSelection()
	if err != nil {
		return err
	}
	if mode == "" {
		return nil
	}

	worktree := domain.Worktree{
		Path:       selection.SelectedPath,
		BranchName: selection.SelectedBranch,
	}
	return worktreeService.Delete(ctx, workspace, worktree, bakerworktree.DeleteMode(mode), true)
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

func promptCreateMode() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("create worktree")
	fmt.Println("1. existing branch")
	fmt.Println("2. new branch from base branch")
	fmt.Println("q. cancel")
	fmt.Print("> ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	switch strings.TrimSpace(input) {
	case "1":
		return "existing", nil
	case "2":
		return "new", nil
	default:
		return "", nil
	}
}

func promptText(label string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s: ", label)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func runBranchSelection(branches []string, includeNewBranchOption bool) (string, error) {
	items := make([]string, 0, len(branches)+1)
	if includeNewBranchOption {
		items = append(items, ui.NewBranchOption)
	}
	items = append(items, branches...)

	finalModel, err := tea.NewProgram(ui.NewModel(ui.State{Screen: ui.ScreenCreateWorktree, Branches: items})).Run()
	if err != nil {
		return "", err
	}
	selected, ok := finalModel.(ui.Model)
	if !ok {
		return "", fmt.Errorf("unexpected ui model type %T", finalModel)
	}
	if selected.SelectedAction == "create-new-branch" {
		return ui.NewBranchOption, nil
	}
	return selected.SelectedBranch, nil
}

func runDeleteModeSelection() (string, error) {
	modes := []string{string(bakerworktree.DeleteModeLocalBranch), string(bakerworktree.DeleteModeAll)}
	finalModel, err := tea.NewProgram(ui.NewModel(ui.State{Screen: ui.ScreenDeleteConfirm, DeleteModes: modes})).Run()
	if err != nil {
		return "", err
	}
	selected, ok := finalModel.(ui.Model)
	if !ok {
		return "", fmt.Errorf("unexpected ui model type %T", finalModel)
	}
	return selected.SelectedAction, nil
}

func findWorkspace(registry config.Registry, workspaceName string) (domain.Workspace, bool) {
	for _, workspace := range registry.Workspaces {
		if workspace.Name == workspaceName {
			return workspace, true
		}
	}
	return domain.Workspace{}, false
}

func worktreeNameForBranch(branch string) string {
	return strings.NewReplacer("/", "-", `\\`, "-").Replace(branch)
}

func loadWorktreeItems(ctx context.Context, paths config.Paths, registry config.Registry, gitClient bakergit.Client) ([]ui.WorktreeItem, error) {
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

		gitWorktrees, err := gitClient.ListWorktrees(ctx, workspace.RepositoryPath)
		if err != nil {
			if os.IsNotExist(err) {
				gitWorktrees = nil
			} else {
				return nil, err
			}
		}

		items := make([]ui.WorktreeItem, 0, len(gitWorktrees)+1)
		for _, worktree := range gitWorktrees {
			if !strings.HasPrefix(filepath.Clean(worktree.Path), filepath.Clean(workspaceRoot)+string(filepath.Separator)) {
				continue
			}
			items = append(items, ui.WorktreeItem{
				Label:         worktreeLabel(worktree.Path),
				Path:          worktree.Path,
				WorkspaceName: workspace.Name,
				WorktreeName:  filepath.Base(worktree.Path),
				BranchName:    worktree.BranchName,
				Selectable:    true,
			})
		}

		sort.Slice(items, func(i, j int) bool { return items[i].Label < items[j].Label })
		groups = append(groups, workspaceItems{name: workspace.Name, items: items})
	}

	sort.Slice(groups, func(i, j int) bool { return groups[i].name < groups[j].name })

	var worktrees []ui.WorktreeItem
	for _, group := range groups {
		worktrees = append(worktrees, ui.WorktreeItem{Label: group.name, WorkspaceName: group.name})
		worktrees = append(worktrees, group.items...)
	}
	return worktrees, nil
}

func worktreeLabel(path string) string {
	return "  " + filepath.Base(path)
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
