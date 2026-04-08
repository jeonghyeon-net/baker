package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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

const (
	githubRepositoryListTimeout = 60 * time.Second
	workspaceCreateTimeout      = 5 * time.Minute
	workspaceSyncTimeout        = 30 * time.Second
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

		selection, err := runWorktreeSelection(ctx, worktrees, registry, githubClient)
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
		case "open-pr-worktree":
			path, err := openPullRequestWorktree(ctx, registry, selection.SelectedWorkspace, selection.SelectedBranch, selection.SelectedPath, gitClient, workspaceService, worktreeService)
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
		case "delete-workspace":
			if err := deleteSelectedWorkspace(paths, registry, selection.SelectedWorkspace, workspaceService); err != nil {
				return "", err
			}
			continue
		default:
			return "", nil
		}
	}
}

func runWorktreeSelection(ctx context.Context, worktrees []ui.WorktreeItem, registry config.Registry, githubClient bakergithub.Client) (ui.Model, error) {
	program := tea.NewProgram(ui.NewModel(ui.State{Screen: ui.ScreenWorktrees, Worktrees: worktrees}), tea.WithAltScreen())

	loadCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, workspace := range registry.Workspaces {
		if workspace.Owner == "" || workspace.Repo == "" {
			continue
		}
		workspace := workspace
		branchPaths := branchPathsForWorkspace(worktrees, workspace.Name)
		go func() {
			prs, err := githubClient.ListMyPullRequestsForRepository(loadCtx, workspace.Owner, workspace.Repo)
			if err != nil {
				program.Send(ui.WorkspacePullRequestsLoadedMsg{WorkspaceName: workspace.Name})
				return
			}
			program.Send(ui.WorkspacePullRequestsLoadedMsg{WorkspaceName: workspace.Name, Items: buildPullRequestItems(workspace.Name, prs, branchPaths)})
		}()
	}

	finalModel, err := program.Run()
	if err != nil {
		return ui.Model{}, err
	}

	selected, ok := finalModel.(ui.Model)
	if !ok {
		return ui.Model{}, fmt.Errorf("예상하지 못한 UI 모델 타입입니다: %T", finalModel)
	}
	return selected, nil
}

type optionChoice struct {
	Label string
	Value string
}

func addWorkspace(ctx context.Context, paths config.Paths, registry config.Registry, githubClient bakergithub.Client, workspaceService bakerworkspace.Service) error {
	mode, err := promptAddWorkspaceMode()
	if err != nil || mode == "" {
		return err
	}

	switch mode {
	case "github":
		ownersCtx, cancel := context.WithTimeout(ctx, githubRepositoryListTimeout)
		defer cancel()
		owners, err := ui.RunStatusValue("불러오는 중", "GitHub 소유자 목록", "GitHub 소유자 목록을 불러오고 있습니다...", func() ([]string, error) {
			return githubClient.ListOwners(ownersCtx)
		})
		if err != nil {
			if errors.Is(ownersCtx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("GitHub 소유자 목록을 %s 안에 불러오지 못했습니다", githubRepositoryListTimeout)
			}
			return err
		}
		if len(owners) == 0 {
			fmt.Println("표시할 GitHub 소유자가 없습니다.")
			return nil
		}

		selectedOwner, err := runOptionSelection("소유자 선택", "enter 선택 • esc 취소", owners)
		if err != nil || selectedOwner == "" {
			return err
		}

		reposCtx, cancelRepos := context.WithTimeout(ctx, githubRepositoryListTimeout)
		defer cancelRepos()
		repos, err := ui.RunStatusValue("불러오는 중", "GitHub 저장소 목록", selectedOwner+" 저장소를 불러오고 있습니다...", func() ([]domain.GitHubRepo, error) {
			return githubClient.ListRepositoriesForOwner(reposCtx, selectedOwner)
		})
		if err != nil {
			if errors.Is(reposCtx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("%s 저장소 목록을 %s 안에 불러오지 못했습니다", selectedOwner, githubRepositoryListTimeout)
			}
			return err
		}
		if len(repos) == 0 {
			fmt.Printf("%s에 표시할 저장소가 없습니다.\n", selectedOwner)
			return nil
		}

		selectedRepo, err := runRepositorySelection(repos)
		if err != nil || selectedRepo == nil {
			return err
		}

		createCtx, cancelCreate := context.WithTimeout(ctx, workspaceCreateTimeout)
		defer cancelCreate()
		updatedRegistry, err := ui.RunStatusValue("불러오는 중", "워크스페이스 추가", selectedRepo.NameWithOwner+" 워크스페이스를 만들고 있습니다...", func() (config.Registry, error) {
			_, updatedRegistry, err := ensureWorkspace(createCtx, paths, registry, workspaceService, *selectedRepo)
			return updatedRegistry, err
		})
		if err != nil {
			if errors.Is(createCtx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("워크스페이스 추가를 %s 안에 끝내지 못했습니다", workspaceCreateTimeout)
			}
			return err
		}
		if err := config.SaveRegistry(paths.RegistryFile, updatedRegistry); err != nil {
			return err
		}
		return nil
	case "url":
		remoteURL, err := promptText("원격 저장소 URL")
		if err != nil || remoteURL == "" {
			return err
		}
		workspaceName := suggestedWorkspaceNameFromRemote(remoteURL)
		if workspaceName == "" {
			workspaceName = strings.ReplaceAll(strings.TrimSuffix(filepath.Base(remoteURL), ".git"), "/", "-")
		}
		createCtx, cancelCreate := context.WithTimeout(ctx, workspaceCreateTimeout)
		defer cancelCreate()
		workspace, err := ui.RunStatusValue("불러오는 중", "워크스페이스 추가", workspaceName+" 워크스페이스를 만들고 있습니다...", func() (domain.Workspace, error) {
			return workspaceService.CreateFromRemoteURL(createCtx, remoteURL, workspaceName)
		})
		if err != nil {
			if errors.Is(createCtx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("워크스페이스 추가를 %s 안에 끝내지 못했습니다", workspaceCreateTimeout)
			}
			return err
		}
		registry.Workspaces = append(registry.Workspaces, workspace)
		if err := config.SaveRegistry(paths.RegistryFile, registry); err != nil {
			return err
		}
		return nil
	default:
		return nil
	}
}

func createWorktreeForWorkspace(ctx context.Context, paths config.Paths, registry config.Registry, workspaceName string, gitClient bakergit.Client, workspaceService bakerworkspace.Service, worktreeService bakerworktree.Service) (string, error) {
	workspace, ok := findWorkspace(registry, workspaceName)
	if !ok {
		return "", fmt.Errorf("워크스페이스를 찾을 수 없습니다: %s", workspaceName)
	}

	mode, err := promptCreateMode()
	if err != nil {
		return "", err
	}
	if mode == "" {
		return "", nil
	}

	syncCtx, cancel := context.WithTimeout(ctx, workspaceSyncTimeout)
	defer cancel()
	if err := ui.RunStatus("불러오는 중", "브랜치 목록", workspaceName+" 브랜치를 불러오고 있습니다...", func() error {
		return workspaceService.Sync(syncCtx, workspace)
	}); err != nil {
		if errors.Is(syncCtx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("%s 브랜치를 %s 안에 불러오지 못했습니다", workspaceName, workspaceSyncTimeout)
		}
		return "", err
	}
	branches, err := gitClient.ListBranches(ctx, workspace.RepositoryPath)
	if err != nil {
		return "", err
	}
	if len(branches) == 0 {
		return "", fmt.Errorf("%s에 사용할 브랜치가 없습니다", workspaceName)
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
	newBranch, err := promptText("새 브랜치 이름")
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

func openPullRequestWorktree(ctx context.Context, registry config.Registry, workspaceName, branchName, existingPath string, gitClient bakergit.Client, workspaceService bakerworkspace.Service, worktreeService bakerworktree.Service) (string, error) {
	if existingPath != "" {
		return existingPath, nil
	}
	workspace, ok := findWorkspace(registry, workspaceName)
	if !ok {
		return "", fmt.Errorf("워크스페이스를 찾을 수 없습니다: %s", workspaceName)
	}

	syncCtx, cancel := context.WithTimeout(ctx, workspaceSyncTimeout)
	defer cancel()
	if err := ui.RunStatus("불러오는 중", "PR 워크트리", branchName+" 브랜치를 준비하고 있습니다...", func() error {
		return workspaceService.Sync(syncCtx, workspace)
	}); err != nil {
		if errors.Is(syncCtx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("%s 브랜치를 %s 안에 준비하지 못했습니다", branchName, workspaceSyncTimeout)
		}
		return "", err
	}

	worktrees, err := gitClient.ListWorktrees(ctx, workspace.RepositoryPath)
	if err == nil {
		for _, worktree := range worktrees {
			if worktree.BranchName == branchName {
				return worktree.Path, nil
			}
		}
	}

	branches, err := gitClient.ListBranches(ctx, workspace.RepositoryPath)
	if err != nil {
		return "", err
	}
	for _, branch := range branches {
		if branch.Name == branchName {
			result, err := worktreeService.CreateFromExistingBranch(ctx, workspace, branches, branchName, worktreeNameForBranch(branchName))
			if err != nil {
				return "", err
			}
			return result.Path, nil
		}
	}

	return "", fmt.Errorf("PR 브랜치를 찾을 수 없습니다: %s", branchName)
}

func deleteSelectedWorktree(ctx context.Context, paths config.Paths, registry config.Registry, selection ui.Model, worktreeService bakerworktree.Service) error {
	workspace, ok := findWorkspace(registry, selection.SelectedWorkspace)
	if !ok {
		return fmt.Errorf("워크스페이스를 찾을 수 없습니다: %s", selection.SelectedWorkspace)
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

func deleteSelectedWorkspace(paths config.Paths, registry config.Registry, workspaceName string, workspaceService bakerworkspace.Service) error {
	workspace, ok := findWorkspace(registry, workspaceName)
	if !ok {
		return fmt.Errorf("워크스페이스를 찾을 수 없습니다: %s", workspaceName)
	}

	choice, err := runMappedOptionSelection("워크스페이스 삭제", "enter 선택 • esc 취소", []optionChoice{{Label: "취소", Value: "cancel"}, {Label: "워크스페이스 삭제", Value: "delete-workspace"}})
	if err != nil {
		return err
	}
	if choice != "delete-workspace" {
		return nil
	}
	if err := workspaceService.Delete(workspace); err != nil {
		return err
	}

	updatedRegistry := config.Registry{Workspaces: make([]domain.Workspace, 0, len(registry.Workspaces))}
	for _, candidate := range registry.Workspaces {
		if candidate.Name == workspaceName {
			continue
		}
		updatedRegistry.Workspaces = append(updatedRegistry.Workspaces, candidate)
	}
	return config.SaveRegistry(paths.RegistryFile, updatedRegistry)
}

func runRepositorySelection(repos []domain.GitHubRepo) (*domain.GitHubRepo, error) {
	names := make([]string, 0, len(repos))
	for _, repo := range repos {
		names = append(names, repo.NameWithOwner)
	}

	finalModel, err := tea.NewProgram(ui.NewModel(ui.State{Screen: ui.ScreenWorkspaceGitHubPicker, Title: "저장소 선택", Hint: "enter 선택 • esc 취소", Repositories: names}), tea.WithAltScreen()).Run()
	if err != nil {
		return nil, err
	}

	selected, ok := finalModel.(ui.Model)
	if !ok {
		return nil, fmt.Errorf("예상하지 못한 UI 모델 타입입니다: %T", finalModel)
	}
	if selected.SelectedPath == "" {
		return nil, nil
	}

	for _, repo := range repos {
		if repo.NameWithOwner == selected.SelectedPath {
			return &repo, nil
		}
	}
	return nil, fmt.Errorf("선택한 저장소를 찾을 수 없습니다: %s", selected.SelectedPath)
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

func runOptionSelection(title, hint string, options []string) (string, error) {
	finalModel, err := tea.NewProgram(ui.NewModel(ui.State{Screen: ui.ScreenOptions, Title: title, Hint: hint, Options: options}), tea.WithAltScreen()).Run()
	if err != nil {
		return "", err
	}
	selected, ok := finalModel.(ui.Model)
	if !ok {
		return "", fmt.Errorf("예상하지 못한 UI 모델 타입입니다: %T", finalModel)
	}
	return selected.SelectedAction, nil
}

func runMappedOptionSelection(title, hint string, options []optionChoice) (string, error) {
	labels := make([]string, 0, len(options))
	byLabel := make(map[string]string, len(options))
	for _, option := range options {
		labels = append(labels, option.Label)
		byLabel[option.Label] = option.Value
	}

	selectedLabel, err := runOptionSelection(title, hint, labels)
	if err != nil || selectedLabel == "" {
		return "", err
	}
	return byLabel[selectedLabel], nil
}

func promptAddWorkspaceMode() (string, error) {
	choice, err := runMappedOptionSelection("워크스페이스 추가", "enter 선택 • esc 취소", []optionChoice{{Label: "GitHub 저장소 선택", Value: "github"}, {Label: "원격 저장소 URL 입력", Value: "url"}})
	if err != nil {
		return "", err
	}
	return choice, nil
}

func suggestedWorkspaceNameFromRemote(remoteURL string) string {
	trimmed := strings.TrimPrefix(remoteURL, "git@github.com:")
	trimmed = strings.TrimSuffix(trimmed, ".git")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return ""
	}
	return parts[0] + "-" + parts[1]
}

func promptCreateMode() (string, error) {
	choice, err := runMappedOptionSelection("워크트리 생성", "enter 선택 • esc 취소", []optionChoice{{Label: "기존 브랜치로 생성", Value: "existing"}, {Label: "새 브랜치 만들어 생성", Value: "new"}})
	if err != nil {
		return "", err
	}
	return choice, nil
}

func promptText(label string) (string, error) {
	return ui.PromptText(label, "enter 입력 • esc 취소", label)
}

func runBranchSelection(branches []string, includeNewBranchOption bool) (string, error) {
	items := make([]string, 0, len(branches)+1)
	if includeNewBranchOption {
		items = append(items, ui.NewBranchOption)
	}
	items = append(items, branches...)

	finalModel, err := tea.NewProgram(ui.NewModel(ui.State{Screen: ui.ScreenCreateWorktree, Title: "브랜치 선택", Hint: "enter 선택 • esc 취소", Branches: items}), tea.WithAltScreen()).Run()
	if err != nil {
		return "", err
	}
	selected, ok := finalModel.(ui.Model)
	if !ok {
		return "", fmt.Errorf("예상하지 못한 UI 모델 타입입니다: %T", finalModel)
	}
	if selected.SelectedAction == "create-new-branch" {
		return ui.NewBranchOption, nil
	}
	return selected.SelectedBranch, nil
}

func runDeleteModeSelection() (string, error) {
	choice, err := runMappedOptionSelection("워크트리 삭제", "enter 선택 • esc 취소", []optionChoice{{Label: "워크트리 + 로컬 브랜치 삭제", Value: string(bakerworktree.DeleteModeLocalBranch)}, {Label: "워크트리 + 로컬 브랜치 + 원격 브랜치 삭제", Value: string(bakerworktree.DeleteModeAll)}})
	if err != nil {
		return "", err
	}
	return choice, nil
}

func branchPathsForWorkspace(items []ui.WorktreeItem, workspaceName string) map[string]string {
	paths := make(map[string]string)
	for _, item := range items {
		if item.WorkspaceName != workspaceName || item.BranchName == "" || item.Path == "" {
			continue
		}
		paths[item.BranchName] = item.Path
	}
	return paths
}

func buildPullRequestItems(workspaceName string, prs []domain.GitHubPullRequest, branchPaths map[string]string) []ui.WorktreeItem {
	items := make([]ui.WorktreeItem, 0, len(prs))
	for i, pr := range prs {
		status := pullRequestStatusLabel(pr)
		items = append(items, ui.WorktreeItem{
			Label:             pullRequestLabel(pr.Number, pr.Title, status, i == len(prs)-1),
			Path:              branchPaths[pr.HeadRefName],
			WorkspaceName:     workspaceName,
			BranchName:        pr.HeadRefName,
			Selectable:        true,
			PullRequestNumber: pr.Number,
			PullRequestTitle:  pr.Title,
			PullRequestStatus: status,
		})
	}
	return items
}

func pullRequestStatusLabel(pr domain.GitHubPullRequest) string {
	if pr.IsDraft {
		return "초안"
	}

	switch pr.ReviewDecision {
	case "APPROVED":
		return "승인"
	case "CHANGES_REQUESTED":
		return "수정 요청"
	case "REVIEW_REQUIRED":
		return "리뷰 대기"
	default:
		return ""
	}
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
				Path:          worktree.Path,
				WorkspaceName: workspace.Name,
				WorktreeName:  filepath.Base(worktree.Path),
				BranchName:    worktree.BranchName,
				Selectable:    true,
			})
		}

		sort.Slice(items, func(i, j int) bool { return items[i].WorktreeName < items[j].WorktreeName })
		if workspace.Owner != "" && workspace.Repo != "" {
			items = append(items, ui.WorktreeItem{WorkspaceName: workspace.Name, PullRequestLoading: true})
		}
		groups = append(groups, workspaceItems{name: workspace.Name, items: items})
	}

	sort.Slice(groups, func(i, j int) bool { return groups[i].name < groups[j].name })

	var worktrees []ui.WorktreeItem
	for _, group := range groups {
		worktrees = append(worktrees, ui.WorktreeItem{Label: "▾ " + group.name, WorkspaceName: group.name})
		for _, item := range relabelGroupItems(group.items) {
			worktrees = append(worktrees, item)
		}
	}
	return worktrees, nil
}

func relabelGroupItems(items []ui.WorktreeItem) []ui.WorktreeItem {
	relabeled := append([]ui.WorktreeItem{}, items...)
	for i := range relabeled {
		last := i == len(relabeled)-1
		switch {
		case relabeled[i].PullRequestLoading:
			relabeled[i].Label = pullRequestLoadingLabel(last)
		case relabeled[i].PullRequestNumber > 0 && relabeled[i].Path == "":
			relabeled[i].Label = pullRequestLabel(relabeled[i].PullRequestNumber, relabeled[i].PullRequestTitle, relabeled[i].PullRequestStatus, last)
		default:
			relabeled[i].Label = worktreeLabel(relabeled[i].WorktreeName, last)
		}
	}
	return relabeled
}

func worktreeLabel(worktreeName string, last bool) string {
	connector := "├─"
	if last {
		connector = "└─"
	}
	return "  " + connector + " " + worktreeName
}

func pullRequestLoadingLabel(last bool) string {
	connector := "├─"
	if last {
		connector = "└─"
	}
	return "  " + connector + " PR 불러오는 중..."
}

func pullRequestLabel(number int, title, status string, last bool) string {
	connector := "├─"
	if last {
		connector = "└─"
	}
	label := fmt.Sprintf("  %s PR #%d %s", connector, number, title)
	if status != "" {
		label += fmt.Sprintf("  [%s]", status)
	}
	return label
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
