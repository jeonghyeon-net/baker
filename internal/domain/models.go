package domain

type Workspace struct {
	Name           string `json:"name"`
	RemoteURL      string `json:"remote_url"`
	Owner          string `json:"owner"`
	Repo           string `json:"repo"`
	DefaultBranch  string `json:"default_branch"`
	RepositoryPath string `json:"repository_path"`
}

type Worktree struct {
	Name          string `json:"name"`
	WorkspaceName string `json:"workspace_name"`
	BranchName    string `json:"branch_name"`
	Path          string `json:"path"`
	HeadSHA       string `json:"head_sha"`
	IsClean       bool   `json:"is_clean"`
	Upstream      string `json:"upstream"`
}

type BranchRef struct {
	Name              string `json:"name"`
	Source            string `json:"source"`
	RemoteName        string `json:"remote_name"`
	ExistsLocally     bool   `json:"exists_locally"`
	HasActiveWorktree bool   `json:"has_active_worktree"`
}

type GitHubRepo struct {
	NameWithOwner string `json:"name_with_owner"`
	SSHURL        string `json:"ssh_url"`
	DefaultBranch string `json:"default_branch"`
}
