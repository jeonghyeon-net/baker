# Baker design spec

Date: 2026-04-08
Status: Draft approved for implementation planning

## 1. Goal

Build `baker`, a terminal application for managing Git workspaces and worktrees with a fast local UX.

Primary goals:
- Launch from anywhere with `baker`
- Manage repositories as **workspaces**
- Manage checked-out branches as **worktrees** under each workspace
- Always operate against the latest remote state
- Let the user select an existing worktree and move the current shell into that directory
- Support creating a worktree either from an existing branch or by creating a new branch from a selected base branch
- Push newly created branches to `origin` immediately

Non-goals for the first version:
- GitHub API direct integration without `gh`
- fish shell support
- advanced Git operations like rebase/merge/stash
- multi-remote workflows
- PR/review management

## 2. Core concepts

### Workspace
A workspace represents a single Git repository.

Examples:
- `git@github.com:jeonghyeon-net/baker.git`
- `git@github.com:org/service.git`

A workspace stores:
- workspace name
- remote URL
- parsed owner/repository name when available
- default branch
- local bare repository path

### Worktree
A worktree is a checked-out branch under a workspace.

Each branch can have at most **one active worktree** managed by Baker.

A worktree stores:
- worktree name
- workspace name
- branch name
- filesystem path
- upstream information
- clean/dirty state
- HEAD SHA

## 3. Filesystem layout

Baker uses consistent full names for storage paths.

- Bare repositories:
  - `~/.pi/repositories/{workspace}`
- Checked out worktrees:
  - `~/.pi/worktrees/{workspace}/{worktree}`

Examples:
- `~/.pi/repositories/jeonghyeon-net-baker`
- `~/.pi/worktrees/jeonghyeon-net-baker/main`
- `~/.pi/worktrees/jeonghyeon-net-baker/feature-login`

Baker-managed config lives under:
- `~/.pi/baker/`

This keeps Baker storage consistent with the repository and worktree paths.

## 4. Technology decisions

- Language: Go
- Runtime/tooling management: `mise`
- Target Go version for implementation: `1.26.0`
- Interface: terminal TUI
- Git operations: shell out to `git`
- GitHub repository discovery: shell out to `gh`

Reasoning:
- Go is fast enough and simpler than Rust for this tool
- `git worktree` behavior should come from Git itself rather than reimplementation
- `gh` already carries the user’s authenticated GitHub context

## 5. Shell integration

### Problem
A child process cannot directly change the current working directory of the parent shell.

### Decision
Baker will use a shell integration hook.

### Supported shells in v1
- zsh
- bash

### First-run behavior
When `baker` is launched without the shell hook installed:
1. Detect the current shell
2. Install the Baker hook into the correct shell rc file
3. Print a message telling the user to run:
   - `source ~/.zshrc` or
   - `source ~/.bashrc`
4. Exit without trying to change the current shell directory

### Subsequent behavior
After the hook is active:
- the shell wrapper invokes the Baker binary
- the Baker binary returns the selected path in a machine-readable form
- the wrapper performs `cd` in the current shell

### Shell hook requirements
- Append a clearly marked Baker block to the rc file
- Do not duplicate the block on repeated runs
- Do not modify unrelated user config
- Fail safely with a clear message if the rc file is not writable

## 6. User-facing flows

### 6.1 Launch
Command:
- `baker`

The app opens a TUI for selecting or managing workspaces and worktrees.

### 6.2 Create workspace
Two entry points are required.

#### A. Manual remote URL input
User enters a Git remote URL, for example:
- `git@github.com:jeonghyeon-net/baker.git`

Baker will:
1. Validate and parse the remote URL
2. Suggest a workspace name using `owner-repo`
3. Allow the user to edit the proposed workspace name
4. Create a bare clone at `~/.pi/repositories/{workspace}`
5. Detect the default branch
6. Register the workspace for later display

#### B. GitHub repository picker
User chooses a repository from a TUI list backed by `gh`.

Baker will:
1. Query accessible repositories via `gh`
2. Show them in a keyboard-navigable list
3. Use the selected repository clone URL
4. Suggest a workspace name using `owner-repo`
5. Allow the user to edit the proposed workspace name
6. Create the bare clone and register the workspace

### 6.3 Enter workspace
Whenever a workspace is opened, Baker must refresh remote state first.

Required sync step:
- `git fetch --all --prune`

The workspace view must display:
- existing worktrees
- local branches
- remote branches not yet present locally
- whether a branch is already checked out by another worktree

### 6.4 Create worktree from existing branch
User flow:
1. Open a workspace
2. Choose “create worktree”
3. Select an existing branch from the unified branch list
4. Provide or confirm a worktree name

Behavior:
- If the branch already has an active worktree, Baker blocks creation
- If the branch exists only on remote, Baker creates/tracks the local branch as needed
- Baker creates the worktree under `~/.pi/worktrees/{workspace}/{worktree}`
- On success, Baker returns that path so the shell wrapper can `cd` into it

### 6.5 Create worktree with a new branch
User flow:
1. Open a workspace
2. Choose “create worktree”
3. Select a base branch
4. Enter a new branch name
5. Confirm or edit the default worktree name

Behavior:
- Sync remote state first
- Create the worktree and branch from the chosen base branch
- Immediately push the new branch to `origin`
- Set upstream tracking
- Only treat the flow as fully successful after the push succeeds
- On success, return the new worktree path for shell `cd`

If push fails:
- clearly show that the local branch/worktree was created but remote push failed
- offer a retry or cleanup path
- do not silently roll forward as if the operation succeeded

### 6.6 Select existing worktree
User flow:
1. Open a workspace
2. Select an existing worktree

Behavior:
- Baker returns the absolute selected path
- The shell integration changes the current shell directory to that path

### 6.7 Delete worktree
Deletion must be interactive and mode-based.

Deletion options:
1. Delete worktree only
2. Delete worktree and local branch
3. Delete worktree, local branch, and remote branch

Required warnings before deletion:
- clean vs dirty status
- current branch name
- upstream presence
- remote deletion implications

Dirty worktrees require stronger confirmation.

### 6.8 Delete workspace
Workspace deletion is a separate destructive action.

Behavior:
- strong confirmation required
- remove the bare repository
- optionally remove all worktrees under that workspace
- clean up any Baker metadata

## 7. Domain and naming rules

### Workspace naming
Default suggestion:
- `owner-repo`

Examples:
- `jeonghyeon-net-baker`
- `acme-payments-service`

The user may override the suggestion.

Workspace names must be filesystem-safe.

### Worktree naming
The user can supply a name, but Baker may propose one based on branch name.

Examples:
- branch `main` -> worktree `main`
- branch `feature/login` -> worktree `feature-login`

Worktree names must be filesystem-safe.

### Branch uniqueness policy
Baker enforces:
- one active worktree per branch per workspace

This applies whether the branch originally came from local or remote.

## 8. Data model

Suggested internal models:

```text
Workspace
- Name
- RemoteURL
- Owner
- Repo
- DefaultBranch
- RepositoryPath

Worktree
- Name
- WorkspaceName
- BranchName
- Path
- HeadSHA
- IsClean
- Upstream

BranchRef
- Name
- Source (local | remote)
- RemoteName
- ExistsLocally
- HasActiveWorktree
```

The implementation may refine fields, but these concepts must remain explicit.

## 9. Architecture

Suggested module boundaries:

- `internal/app`
  - startup, dependency wiring, preflight checks
- `internal/ui`
  - TUI screens, key handling, selection flows, confirmations
- `internal/git`
  - `git` command wrappers and output parsing
- `internal/github`
  - `gh` command wrappers for repo discovery
- `internal/workspace`
  - workspace creation, listing, sync, deletion
- `internal/worktree`
  - worktree creation, selection, deletion
- `internal/shell`
  - shell detection, hook installation, wrapper protocol
- `internal/config`
  - local config, paths, persistence, hook metadata

Guiding rules:
- UI should not contain raw Git command orchestration
- Git/GitHub access should be encapsulated behind small service interfaces
- shell integration must remain isolated from worktree business logic

## 10. Remote synchronization policy

Baker is remote-aware by default.

Required behaviors:
- fetch on workspace entry
- include remote-only branches in branch selection
- prune deleted remote refs during refresh
- push newly created branches immediately
- display stale/error state when refresh fails

Baker must not pretend remote state is fresh if fetch failed.

## 11. Error handling

Baker should expose failures clearly without dumping unreadable noise by default.

Principles:
- show which operation failed
- provide a concise summary first
- allow viewing command details when useful
- do not hide partial states

Examples of important failures:
- `git` is missing
- `gh` is missing or not authenticated
- fetch failed
- clone failed
- worktree path already exists
- branch already has another worktree
- push failed after local branch creation
- rc file is not writable

## 12. Safety rules

Baker should prefer safety over convenience for destructive actions.

Required safeguards:
- block duplicate worktree creation for the same branch
- preflight validation for names and paths
- strong confirmation for destructive deletes
- additional warning for dirty worktree deletion
- remote branch deletion requires explicit intent
- no duplicate shell hook insertion

## 13. Preflight checks

At startup or before entering destructive/remote flows, Baker should verify:
- `git` exists
- `gh` exists for GitHub picker flows
- `gh auth status` is valid when needed
- configured directories are writable or creatable
- shell is supported for hook installation

If a prerequisite is missing, Baker should show the fix rather than failing with a raw stack trace.

## 14. Testing strategy

### Unit tests
Cover:
- remote URL parsing
- workspace name generation and normalization
- worktree name normalization
- shell detection
- rc file hook insertion and idempotency
- branch list merging logic
- command output parsing helpers

### Integration tests
Use temporary Git repositories and real `git` commands to verify:
- bare repository setup
- fetch/prune behavior
- worktree add/remove
- tracking a remote-only branch
- creating a new branch from a base branch
- push with upstream setup
- duplicate worktree prevention

### End-to-end tests
Validate user-level flows such as:
- first run installs hook and exits with instructions
- workspace creation from URL
- workspace creation from `gh` picker
- worktree creation from existing branch
- worktree creation with new branch plus push
- worktree deletion options
- path handoff to shell wrapper

### Manual validation
Before release, verify interactively in real shells:
- zsh flow
- bash flow
- hook installation idempotency
- same-session activation after `source ~/.zshrc` or `source ~/.bashrc`
- behavior when network or GitHub auth is unavailable

## 15. MVP scope

The first implementation must include:
- `baker` launchable from PATH
- automatic shell hook installation for zsh/bash
- first-run instruction to `source` the shell rc file
- workspace creation from URL input
- workspace creation from GitHub repo picker via `gh`
- remote refresh on workspace entry
- existing worktree selection and shell path handoff
- worktree creation from an existing branch
- worktree creation from a new branch plus push to origin
- interactive worktree deletion with multiple deletion modes
- a top-level plain-text `README` file instead of `README.md`

## 16. Deferred ideas

Possible future additions, explicitly deferred from MVP:
- fish shell support
- background sync/caching
- richer search/filter/sort in very large repo lists
- metadata persistence for usage history and favorites
- PR-aware shortcuts
- branch protection awareness
- opening worktrees in an editor automatically

## 17. Documentation convention

Project documentation for the repository root should follow an old-style plain-text convention:
- use `README`
- do not use `README.md`

Markdown is still acceptable for design/spec documents under `docs/specs/`, but the top-level project readme should be plain text.

## 18. Open implementation notes

These are not unresolved product questions; they are implementation details to settle during planning:
- exact TUI library choice in Go
- command protocol between shell wrapper and binary
- how much command stderr is shown inline vs expandable details

Those decisions must preserve the behavior defined in this spec.
