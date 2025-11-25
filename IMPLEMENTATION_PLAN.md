# Sesh Implementation Plan

## Project Overview

**sesh** is a git workspace and tmux session manager written in Go. It manages project clones and git worktrees in a centralized workspace folder, and creates associated tmux sessions for each workspace.

**Key Features**:
- Manages git repository clones in a user-specified workspace folder (default: `~/.sesh`)
- Creates and manages git worktrees for different branches
- Abstracted session manager backend (initial implementation: tmux)
- Automatically creates sessions tied to worktrees
- Provides fuzzy search for remote branches to switch to or create new worktrees
- Unified switch/create command (similar to `git checkout` vs `git checkout -b`)
- Auto-detects project from current working directory when not specified
- Extensible architecture for multiple session manager backends (tmux, zellij, screen, etc.)

**Current Status**: Project bootstrapped with Cobra CLI framework, Taskfile.yaml for build automation, and AGENTS.md for development guidelines.

**Goal**: Build a fully functional workspace manager that seamlessly integrates git worktrees with tmux sessions.

---

## Phase 1: Foundation (Core Infrastructure)

### 1.1 Project Structure Setup

**Objective**: Create internal package structure for organizing code.

**Tasks**:
- [ ] Create `internal/config/` directory
- [ ] Create `internal/db/` directory with `migrations/` subdirectory
- [ ] Create `internal/session/` directory
- [ ] Create `internal/workspace/` directory
- [ ] Create `internal/models/` directory

**Expected Structure**:
```
internal/
├── config/         # Configuration and path management
│   └── config.go
├── db/             # Database layer and migrations
│   ├── db.go
│   ├── migrations.go
│   └── migrations/
│       └── 001_initial_schema.sql
├── git/            # Git operations (clone, fetch, worktree)
│   ├── clone.go
│   ├── worktree.go
│   └── branch.go
├── session/        # Session manager abstraction
│   ├── manager.go      # Interface and factory
│   ├── tmux.go         # Tmux backend implementation
│   └── backends/       # Future backends (zellij, screen, etc.)
├── workspace/      # Workspace folder management
│   └── workspace.go
├── project/        # Project resolution and detection
│   └── project.go
├── fuzzy/          # Fuzzy finder integration
│   └── fuzzy.go
└── models/         # Data models
    └── models.go
```

**Estimated Time**: 15 minutes

---

### 1.2 Configuration Management

**Objective**: Implement configuration system to manage application paths and settings.

**Location**: `internal/config/config.go`

**Tasks**:
- [ ] Implement `GetConfigDir()` - Returns OS-specific config directory + `/sesh`
- [ ] Implement `GetWorkspaceDir()` - Returns workspace folder (default: `~/.sesh`, configurable)
- [ ] Implement `GetSessionBackend()` - Returns session backend name (default: auto-detect, configurable)
- [ ] Implement `GetDBPath()` - Returns full path to SQLite database
- [ ] Implement `EnsureConfigDir()` - Creates config directory if it doesn't exist
- [ ] Implement `EnsureWorkspaceDir()` - Creates workspace directory if it doesn't exist
- [ ] Support config file for settings override (`~/.config/sesh/config.yaml`)
- [ ] Support environment variables for configuration override
- [ ] Handle permissions and errors using eris
- [ ] Add unit tests for path resolution

**Key Functions**:
```go
type Config struct {
    WorkspaceDir   string
    SessionBackend string  // "tmux", "zellij", "screen", "auto"
    // Future: FuzzyFinder, DefaultBranch, etc.
}

func GetConfigDir() (string, error)
func GetWorkspaceDir() (string, error)
func GetSessionBackend() (string, error)
func GetDBPath() (string, error)
func EnsureConfigDir() error
func EnsureWorkspaceDir() error
func LoadConfig() (*Config, error)
```

**Configuration Hierarchy** (highest to lowest priority):

Workspace Directory:
1. Environment variable: `$SESH_WORKSPACE`
2. Config file: `~/.config/sesh/config.yaml` → `workspace_dir`
3. Default: `~/.sesh`

Session Backend:
1. Environment variable: `$SESH_SESSION_BACKEND`
2. Config file: `~/.config/sesh/config.yaml` → `session_backend`
3. Auto-detect: tmux → zellij → screen → error

**Example config.yaml**:
```yaml
workspace_dir: ~/Code/workspaces
session_backend: tmux  # or "zellij", "screen", "auto"
```

**Platform-specific config paths**:
- Linux: `$XDG_CONFIG_HOME/sesh` or `~/.config/sesh`
- macOS: `~/Library/Application Support/sesh`
- Windows: `%APPDATA%/sesh`

**Workspace folder structure**:
```
~/.sesh/
├── github.com/
│   └── user/
│       └── repo/
│           ├── main/              # Main worktree
│           ├── feature-branch/    # Worktree for feature-branch
│           └── .git/              # Bare repository
└── gitlab.com/
    └── org/
        └── project/
```

**Estimated Time**: 2-3 hours

---

### 1.3 Database Schema Design

**Objective**: Design database schema for projects, worktrees, and sessions.

**Location**: `internal/db/migrations/001_initial_schema.sql`

**Schema**:

```sql
-- projects table (git repositories)
CREATE TABLE IF NOT EXISTS projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,           -- e.g., "github.com/user/repo"
    remote_url TEXT NOT NULL,            -- Git remote URL
    local_path TEXT NOT NULL,            -- Path to bare repo in workspace
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_fetched DATETIME,
    UNIQUE(remote_url)
);

CREATE INDEX idx_projects_name ON projects(name);

-- worktrees table (git worktrees for different branches)
CREATE TABLE IF NOT EXISTS worktrees (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    branch TEXT NOT NULL,                -- Branch/ref name
    path TEXT UNIQUE NOT NULL,           -- Path to worktree
    is_main BOOLEAN DEFAULT 0,           -- Is this the main worktree?
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_used DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    UNIQUE(project_id, branch)
);

CREATE INDEX idx_worktrees_project_id ON worktrees(project_id);
CREATE INDEX idx_worktrees_branch ON worktrees(branch);
CREATE INDEX idx_worktrees_path ON worktrees(path);

-- sessions table (tmux sessions tied to worktrees)
CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    worktree_id INTEGER NOT NULL,
    tmux_session_name TEXT UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_attached DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (worktree_id) REFERENCES worktrees(id) ON DELETE CASCADE
);

CREATE INDEX idx_sessions_worktree_id ON sessions(worktree_id);
CREATE INDEX idx_sessions_tmux_name ON sessions(tmux_session_name);

-- schema_migrations for tracking applied migrations
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**Data Model Relationships**:
```
Project (1) ──→ (N) Worktrees (1) ──→ (1) Session
  ↓                    ↓                    ↓
repo clone        branch worktree      tmux session
```

**Estimated Time**: 45 minutes

---

### 1.4 Database Layer Implementation

**Objective**: Implement database connection, migrations, and CRUD operations.

**Location**: `internal/db/db.go` and `internal/db/migrations.go`

**Tasks**:
- [ ] Add SQLite dependency: `go get modernc.org/sqlite` (pure Go, recommended)
- [ ] Implement `InitDB()` - Initialize database connection
- [ ] Implement `RunMigrations()` - Execute SQL migrations
- [ ] Implement `Close()` - Close database connection
- [ ] Create project CRUD operations:
  - [ ] `CreateProject(project *models.Project)`
  - [ ] `GetProject(name string)`
  - [ ] `GetProjectByRemote(remoteURL string)`
  - [ ] `GetAllProjects()`
  - [ ] `UpdateProjectFetchTime(id int)`
  - [ ] `DeleteProject(id int)`
- [ ] Create worktree CRUD operations:
  - [ ] `CreateWorktree(worktree *models.Worktree)`
  - [ ] `GetWorktree(projectID int, branch string)`
  - [ ] `GetWorktreeByPath(path string)`
  - [ ] `GetWorktreesByProject(projectID int)`
  - [ ] `UpdateWorktreeLastUsed(id int)`
  - [ ] `DeleteWorktree(id int)`
- [ ] Create session CRUD operations:
  - [ ] `CreateSession(session *models.Session)`
  - [ ] `GetSessionByWorktree(worktreeID int)`
  - [ ] `GetSessionByTmuxName(tmuxName string)`
  - [ ] `GetAllSessions()`
  - [ ] `UpdateSessionLastAttached(id int)`
  - [ ] `DeleteSession(id int)`
- [ ] Add database tests with in-memory SQLite

**Key Functions**:
```go
func InitDB(dbPath string) (*sql.DB, error)
func RunMigrations(db *sql.DB) error

// Projects
func CreateProject(db *sql.DB, project *models.Project) error
func GetProject(db *sql.DB, name string) (*models.Project, error)
func GetProjectByRemote(db *sql.DB, remoteURL string) (*models.Project, error)

// Worktrees
func CreateWorktree(db *sql.DB, worktree *models.Worktree) error
func GetWorktree(db *sql.DB, projectID int, branch string) (*models.Worktree, error)
func GetWorktreesByProject(db *sql.DB, projectID int) ([]*models.Worktree, error)

// Sessions
func CreateSession(db *sql.DB, session *models.Session) error
func GetSessionByWorktree(db *sql.DB, worktreeID int) (*models.Session, error)
func GetAllSessions(db *sql.DB) ([]*models.Session, error)
```

**Error Handling**: All errors must be wrapped with eris

**Estimated Time**: 5-7 hours

---

### 1.5 Data Models

**Objective**: Define data structures for projects, worktrees, and sessions.

**Location**: `internal/models/models.go`

**Tasks**:
- [ ] Define `Project` struct
- [ ] Define `Worktree` struct
- [ ] Define `Session` struct
- [ ] Add JSON tags for future serialization
- [ ] Add validation methods if needed
- [ ] Add helper methods for common operations

**Structs**:
```go
type Project struct {
    ID          int       `json:"id"`
    Name        string    `json:"name"`           // e.g., "github.com/user/repo"
    RemoteURL   string    `json:"remote_url"`
    LocalPath   string    `json:"local_path"`
    CreatedAt   time.Time `json:"created_at"`
    LastFetched *time.Time `json:"last_fetched,omitempty"`
}

type Worktree struct {
    ID        int       `json:"id"`
    ProjectID int       `json:"project_id"`
    Branch    string    `json:"branch"`
    Path      string    `json:"path"`
    IsMain    bool      `json:"is_main"`
    CreatedAt time.Time `json:"created_at"`
    LastUsed  time.Time `json:"last_used"`
}

type Session struct {
    ID              int       `json:"id"`
    WorktreeID      int       `json:"worktree_id"`
    TmuxSessionName string    `json:"tmux_session_name"`
    CreatedAt       time.Time `json:"created_at"`
    LastAttached    time.Time `json:"last_attached"`
}

// Composite type for queries
type SessionDetails struct {
    Session  *Session
    Worktree *Worktree
    Project  *Project
}
```

**Estimated Time**: 45 minutes

---

## Phase 2: Core Business Logic

### 2.1 Git Clone Management

**Objective**: Implement git repository cloning and management in workspace folder.

**Location**: `internal/git/clone.go`

**Tasks**:
- [ ] Implement `Clone(remoteURL, destPath string)` - Clone repository as bare repo
- [ ] Implement `GetRemoteURL(repoPath string)` - Get remote URL from repo
- [ ] Implement `ParseRemoteURL(remoteURL string)` - Parse remote URL to extract host/org/repo
- [ ] Implement `GenerateProjectName(remoteURL string)` - Generate project name (e.g., "github.com/user/repo")
- [ ] Implement `Fetch(repoPath string)` - Fetch latest changes from remote
- [ ] Handle authentication (SSH keys, HTTPS tokens)
- [ ] Add tests with temporary bare repos

**Key Functions**:
```go
func Clone(remoteURL, destPath string) error
func GetRemoteURL(repoPath string) (string, error)
func ParseRemoteURL(remoteURL string) (host, org, repo string, error)
func GenerateProjectName(remoteURL string) (string, error)
func Fetch(repoPath string) error
```

**Git Commands**:
- Clone bare: `git clone --bare <remote> <dest>`
- Get remote: `git -C <path> remote get-url origin`
- Fetch: `git -C <path> fetch origin`

**Estimated Time**: 3-4 hours

---

### 2.2 Git Worktree Management

**Objective**: Implement git worktree creation and management.

**Location**: `internal/git/worktree.go`

**Tasks**:
- [ ] Implement `CreateWorktree(repoPath, branch, worktreePath string)` - Create new worktree
- [ ] Implement `CreateWorktreeFromRef(repoPath, ref, worktreePath string)` - Create worktree from specific ref
- [ ] Implement `ListWorktrees(repoPath string)` - List all worktrees for a repo
- [ ] Implement `RemoveWorktree(worktreePath string)` - Remove a worktree
- [ ] Implement `GetWorktreeBranch(worktreePath string)` - Get branch name for worktree
- [ ] Handle worktree conflicts and errors
- [ ] Add tests with temporary worktrees

**Key Functions**:
```go
func CreateWorktree(repoPath, branch, worktreePath string) error
func CreateWorktreeFromRef(repoPath, ref, worktreePath string) error
func ListWorktrees(repoPath string) ([]string, error)
func RemoveWorktree(worktreePath string) error
func GetWorktreeBranch(worktreePath string) (string, error)
```

**Git Commands**:
- Create: `git -C <repo> worktree add <path> <branch>`
- Create from ref: `git -C <repo> worktree add <path> <ref>`
- List: `git -C <repo> worktree list --porcelain`
- Remove: `git -C <repo> worktree remove <path>`
- Get branch: `git -C <path> branch --show-current`

**Estimated Time**: 3-4 hours

---

### 2.3 Git Branch Operations

**Objective**: Implement branch listing and remote branch operations.

**Location**: `internal/git/branch.go`

**Tasks**:
- [ ] Implement `ListLocalBranches(repoPath string)` - List all local branches
- [ ] Implement `ListRemoteBranches(repoPath string)` - List all remote branches
- [ ] Implement `ListAllBranches(repoPath string)` - List both local and remote branches
- [ ] Implement `DoesBranchExist(repoPath, branch string)` - Check if branch exists
- [ ] Implement `GetCurrentBranch(worktreePath string)` - Get current branch in worktree
- [ ] Parse branch output for fuzzy finder compatibility
- [ ] Add tests with temporary repos

**Key Functions**:
```go
func ListLocalBranches(repoPath string) ([]string, error)
func ListRemoteBranches(repoPath string) ([]string, error)
func ListAllBranches(repoPath string) ([]string, error)
func DoesBranchExist(repoPath, branch string) (bool, error)
func GetCurrentBranch(worktreePath string) (string, error)
```

**Git Commands**:
- Local branches: `git -C <repo> branch --format='%(refname:short)'`
- Remote branches: `git -C <repo> branch -r --format='%(refname:short)'`
- Check existence: `git -C <repo> rev-parse --verify <branch>`
- Current branch: `git -C <path> branch --show-current`

**Estimated Time**: 2-3 hours

---

### 2.4 Project Resolution

**Objective**: Resolve project from command arguments or current working directory.

**Location**: `internal/project/project.go`

**Tasks**:
- [ ] Implement `ResolveProject(projectName string, cwd string)` - Resolve project from name or CWD
- [ ] Implement `DetectProjectFromCWD(cwd string)` - Detect project from current directory
- [ ] Implement `FindGitRoot(path string)` - Find git repository root from any path
- [ ] Implement `ExtractProjectFromRemote(remoteURL string)` - Extract project name from remote
- [ ] Handle edge cases (not in git repo, multiple remotes)
- [ ] Add tests with various directory structures

**Key Functions**:
```go
func ResolveProject(projectName string, cwd string, db *sql.DB) (*models.Project, error)
func DetectProjectFromCWD(cwd string) (string, error)
func FindGitRoot(path string) (string, error)
func ExtractProjectFromRemote(remoteURL string) (string, error)
```

**Logic**:
1. If `projectName` provided, look up in database
2. If not provided, detect git repo from CWD
3. Get remote URL from git repo
4. Match remote URL to project in database
5. Return error if not found

**Estimated Time**: 2-3 hours

---

### 2.5 Fuzzy Finder Integration

**Objective**: Integrate fuzzy finder for interactive branch selection.

**Location**: `internal/fuzzy/fuzzy.go`

**Tasks**:
- [ ] Implement `SelectBranch(branches []string)` - Fuzzy select from branch list
- [ ] Detect available fuzzy finder (fzf, peco, etc.)
- [ ] Implement fallback if no fuzzy finder available
- [ ] Support piping branch list to fuzzy finder
- [ ] Parse selected branch from fuzzy finder output
- [ ] Add configuration for preferred fuzzy finder
- [ ] Add tests with mocked fuzzy finder

**Key Functions**:
```go
func SelectBranch(branches []string) (string, error)
func DetectFuzzyFinder() (string, error)
func RunFuzzyFinder(items []string, finder string) (string, error)
```

**Fuzzy Finders** (in order of preference):
1. `fzf` - Most popular
2. `peco` - Alternative
3. Simple numbered list fallback

**Estimated Time**: 2-3 hours

---

### 2.6 Session Manager Abstraction

**Objective**: Create abstraction layer for session manager backends, with initial tmux implementation.

**Location**: `internal/session/`

#### 2.6.1 Session Manager Interface

**Location**: `internal/session/manager.go`

**Tasks**:
- [ ] Define `SessionManager` interface
- [ ] Implement `NewSessionManager(backend string)` factory function
- [ ] Add backend detection logic (auto-detect available backends)
- [ ] Support configuration override for backend selection
- [ ] Add backend registration mechanism for future extensions

**Interface Definition**:
```go
// SessionManager defines the interface that all session backends must implement
type SessionManager interface {
    // Create creates a new session with the given name at the specified path
    Create(name, path string) error

    // Attach attaches to an existing session
    Attach(name string) error

    // Switch switches to a session (used when already inside a session)
    Switch(name string) error

    // List returns all active session names
    List() ([]string, error)

    // Delete deletes/kills a session
    Delete(name string) error

    // Exists checks if a session exists
    Exists(name string) (bool, error)

    // IsRunning checks if the session manager is running/available
    IsRunning() (bool, error)

    // Name returns the backend name (e.g., "tmux", "zellij")
    Name() string
}

// Factory function
func NewSessionManager(backend string) (SessionManager, error)

// Auto-detect available backend
func DetectBackend() (string, error)
```

**Backend Detection Priority**:
1. User configuration (`SESH_SESSION_BACKEND` env var or config file)
2. Auto-detect: tmux → zellij → screen → error

**Estimated Time**: 1-2 hours

---

#### 2.6.2 Tmux Backend Implementation

**Objective**: Implement SessionManager interface for tmux.

**Location**: `internal/session/tmux.go`

**Tasks**:
- [ ] Implement `TmuxManager` struct
- [ ] Implement all `SessionManager` interface methods
- [ ] Handle tmux-specific commands via `os/exec`
- [ ] Detect if currently inside tmux (for Switch vs Attach)
- [ ] Add error handling for tmux not installed
- [ ] Parse tmux output correctly
- [ ] Add tests with mocked exec commands

**Implementation**:
```go
type TmuxManager struct{}

func NewTmuxManager() *TmuxManager {
    return &TmuxManager{}
}

func (t *TmuxManager) Create(name, path string) error
func (t *TmuxManager) Attach(name string) error
func (t *TmuxManager) Switch(name string) error
func (t *TmuxManager) List() ([]string, error)
func (t *TmuxManager) Delete(name string) error
func (t *TmuxManager) Exists(name string) (bool, error)
func (t *TmuxManager) IsRunning() (bool, error)
func (t *TmuxManager) Name() string
```

**Tmux Commands**:
- Create: `tmux new-session -d -s <name> -c <path>`
- Attach: `tmux attach-session -t <name>`
- Switch: `tmux switch-client -t <name>` (when inside tmux)
- List: `tmux list-sessions -F "#{session_name}"`
- Delete: `tmux kill-session -t <name>`
- Exists: `tmux has-session -t <name>` (check exit code)
- Detect inside tmux: Check `$TMUX` environment variable

**Estimated Time**: 3-4 hours

---

#### 2.6.3 Future Backend Implementations

**Potential Backends** (not in MVP):

1. **Zellij** - Modern terminal multiplexer
   - Location: `internal/session/backends/zellij.go`
   - Commands differ from tmux

2. **GNU Screen** - Classic terminal multiplexer
   - Location: `internal/session/backends/screen.go`
   - Different command syntax

3. **Wezterm** - Terminal with built-in multiplexing
   - Location: `internal/session/backends/wezterm.go`
   - Uses different mechanism

4. **No Backend** - Just manage worktrees, no sessions
   - Location: `internal/session/backends/none.go`
   - Useful for users who don't use multiplexers

**Implementation Note**: Each backend would implement the `SessionManager` interface and be registered with the factory.

**Estimated Time**: 2-3 hours per additional backend

---

### 2.7 Workspace Folder Management

**Objective**: Manage workspace folder structure and organization.

**Location**: `internal/workspace/workspace.go`

**Tasks**:
- [ ] Implement `GetProjectPath(workspaceDir, projectName string)` - Get path for project
- [ ] Implement `GetWorktreePath(projectPath, branch string)` - Get path for worktree
- [ ] Implement `EnsureProjectDir(projectPath string)` - Create project directory
- [ ] Implement `GenerateTmuxSessionName(projectName, branch string)` - Generate unique session name
- [ ] Implement path sanitization for branches with special characters
- [ ] Add tests for path generation

**Key Functions**:
```go
func GetProjectPath(workspaceDir, projectName string) string
func GetWorktreePath(projectPath, branch string) string
func EnsureProjectDir(projectPath string) error
func GenerateTmuxSessionName(projectName, branch string) string
func SanitizeBranchName(branch string) string
```

**Path Structure**:
```
~/.sesh/
  github.com/
    user/
      repo/
        .git/              # Bare repository
        main/              # Main branch worktree
        feature-foo/       # Feature branch worktree
```

**Estimated Time**: 2-3 hours

---

## Phase 3: CLI Commands

### 3.1 Clone Command

**Objective**: Clone a git repository into the workspace folder.

**Command**: `sesh clone <remote-url>`

**Tasks**:
- [ ] Create command with: `task cobra:add -- clone`
- [ ] Parse remote URL to generate project name
- [ ] Check if project already exists in database
- [ ] Clone repository as bare repo in workspace folder
- [ ] Create project record in database
- [ ] Create main worktree (default branch)
- [ ] Create tmux session for main worktree
- [ ] Attach to the new session
- [ ] Display success message

**Example Usage**:
```bash
sesh clone git@github.com:user/repo.git
# Clones to ~/.sesh/github.com/user/repo/
# Creates worktree at ~/.sesh/github.com/user/repo/main/
# Creates tmux session "github.com/user/repo:main"
# Attaches to session
```

**Estimated Time**: 3-4 hours

---

### 3.2 Switch Command (Unified Create/Switch)

**Objective**: Switch to a branch (create worktree if needed) and attach to tmux session.

**Command**:
- `sesh switch [project] <branch>` - Switch to existing branch
- `sesh switch [project] -b <new-branch>` - Create new branch and switch

**Tasks**:
- [ ] Create command with: `task cobra:add -- switch`
- [ ] Add `-b` flag for creating new branch (like `git checkout -b`)
- [ ] Resolve project from argument or CWD
- [ ] If no branch specified, show fuzzy finder with all branches
- [ ] Check if worktree exists for branch
- [ ] If worktree exists, attach to existing session
- [ ] If worktree doesn't exist, create it
- [ ] Create tmux session if needed
- [ ] Attach to tmux session
- [ ] Update last_used timestamps

**Example Usage**:
```bash
# From any directory in a project
sesh switch feature-foo          # Switch to feature-foo branch

# With explicit project
sesh switch myproject feature-bar

# Create new branch
sesh switch -b new-feature

# Interactive fuzzy selection
sesh switch                      # Shows fuzzy finder with all branches
```

**Fuzzy Search Behavior**:
- If no branch specified, fetch latest and show all remote branches
- Use fzf/peco for interactive selection
- Create worktree for selected branch if needed

**Estimated Time**: 4-5 hours

---

### 3.3 List Command

**Objective**: Display all projects, worktrees, and sessions.

**Command**: `sesh list [options]`

**Tasks**:
- [ ] Create command with: `task cobra:add -- list`
- [ ] Add `--projects` flag to show only projects
- [ ] Add `--sessions` flag to show only sessions
- [ ] Default: show all sessions with project and branch info
- [ ] Query database with joins
- [ ] Display in table format
- [ ] Handle empty lists gracefully
- [ ] Add `--json` flag for JSON output

**Example Output**:
```
# sesh list
PROJECT                    BRANCH         SESSION NAME                  LAST USED
github.com/user/repo       main           repo:main                    2 hours ago
github.com/user/repo       feature-foo    repo:feature-foo             5 mins ago
gitlab.com/org/project     develop        project:develop              1 day ago

# sesh list --projects
PROJECT                    WORKTREES    LAST FETCHED
github.com/user/repo       2            10 mins ago
gitlab.com/org/project     1            2 days ago
```

**Estimated Time**: 3-4 hours

---

### 3.4 Delete Command

**Objective**: Delete worktree, session, or entire project.

**Command**:
- `sesh delete [project] <branch>` - Delete worktree and session
- `sesh delete [project] --all` - Delete entire project

**Tasks**:
- [ ] Create command with: `task cobra:add -- delete`
- [ ] Add `--all` flag to delete entire project
- [ ] Resolve project from argument or CWD
- [ ] Kill tmux session if running
- [ ] Remove git worktree
- [ ] Delete from database
- [ ] If deleting last worktree, optionally delete project
- [ ] Display confirmation message
- [ ] Add `--force` flag to skip confirmation

**Example Usage**:
```bash
# Delete specific worktree/session
sesh delete feature-foo

# Delete entire project
sesh delete myproject --all

# From within a project directory
sesh delete --all
```

**Estimated Time**: 2-3 hours

---

### 3.5 Status Command

**Objective**: Show current session and project information.

**Command**: `sesh status`

**Tasks**:
- [ ] Create command with: `task cobra:add -- status`
- [ ] Detect current tmux session
- [ ] Resolve project from CWD
- [ ] Display current session info
- [ ] Display worktree info
- [ ] Display project info
- [ ] Show git status summary

**Example Output**:
```
Current Session: repo:feature-foo
Project: github.com/user/repo
Branch: feature-foo
Worktree: ~/.sesh/github.com/user/repo/feature-foo
Git Status: 3 modified, 1 untracked

Other Sessions:
  repo:main (last used 2 hours ago)
```

**Estimated Time**: 2-3 hours

---

### 3.6 Fetch Command

**Objective**: Fetch latest changes from remote for a project.

**Command**: `sesh fetch [project]`

**Tasks**:
- [ ] Create command with: `task cobra:add -- fetch`
- [ ] Resolve project from argument or CWD
- [ ] Run `git fetch` on bare repository
- [ ] Update last_fetched timestamp
- [ ] Display fetch results
- [ ] Support `--all` flag to fetch all projects

**Example Usage**:
```bash
# Fetch current project
sesh fetch

# Fetch specific project
sesh fetch myproject

# Fetch all projects
sesh fetch --all
```

**Estimated Time**: 1-2 hours

---

### 3.7 Additional Commands (Future Enhancements)

**Optional commands to consider**:

- `sesh switch <name>` - Alias for attach
- `sesh info <name>` - Show detailed session information
- `sesh rename <old> <new>` - Rename a session
- `sesh sync` - Sync database with running tmux sessions
- `sesh clean` - Remove stale database entries

**Estimated Time**: 1-2 hours per command

---

## Phase 4: Integration & Testing

### 4.1 Integration

**Objective**: Wire all components together and ensure proper error handling.

**Tasks**:
- [ ] ~Create database connection initialization in root command~ (database has been removed from the plan)
- [ ] Pass database connection to all commands
- [ ] Implement graceful database cleanup on exit
- [ ] Ensure all errors use eris wrapping
- [ ] Add helpful error messages for common issues
- [ ] Test end-to-end workflows

**Estimated Time**: 2-3 hours

---

### 4.2 Unit Testing

**Objective**: Add comprehensive unit tests for all packages.

**Tasks**:
- [ ] Add tests for `internal/config` package
- [ ] Add tests for `internal/db` package (use in-memory SQLite)
- [ ] Add tests for `internal/session` package (mock exec)
- [ ] Add tests for `internal/workspace` package (temporary git repos)
- [ ] Aim for >70% code coverage
- [ ] Run `task test:coverage` to verify

**Estimated Time**: 4-6 hours

---

### 4.3 Integration Testing

**Objective**: Test real-world scenarios with actual tmux and git.

**Tasks**:
- [ ] Create integration test suite
- [ ] Test creating session with real tmux
- [ ] Test attaching to sessions
- [ ] Test deleting sessions
- [ ] Test git workspace detection
- [ ] Add integration tests to CI pipeline (future)

**Estimated Time**: 2-3 hours

---

### 4.4 Quality Checks

**Objective**: Ensure code quality and consistency.

**Tasks**:
- [ ] Run `task fmt` - Format all code
- [ ] Run `task lint` - Check for linting issues
- [ ] Fix any linter warnings
- [ ] Run `task test` - Verify all tests pass
- [ ] Run `task check` - Comprehensive quality check
- [ ] Review AGENTS.md compliance

**Estimated Time**: 1-2 hours

---

## Phase 5: Polish & Documentation

### 5.1 User Experience Enhancements

**Objective**: Improve CLI usability and output formatting.

**Tasks**:
- [ ] Add ability to add a command or script to run on opening a session (e.g. direnv allow)
  - This should be configurable as a flag, globally, and per project
  - implement a yaml config file format that gets stored in the os config dir and/or in a 
    repository root.
- [ ] Add colored output for better readability
  - Consider: `github.com/fatih/color` or `github.com/charmbracelet/lipgloss`
- [ ] Add table formatting for list command
  - Consider: `github.com/olekukonko/tablewriter` or `github.com/charmbracelet/bubbles`
- [ ] Add shell completion scripts (bash, zsh, fish)
  - Use Cobra's built-in completion generation

**Estimated Time**: 3-4 hours

---

### 5.2 Documentation

**Objective**: Create comprehensive user documentation.

**Tasks**:
- [ ] Create detailed README.md with:
  - [ ] Project description
  - [ ] Features list
  - [ ] Installation instructions
  - [ ] Usage examples
  - [ ] Configuration options
  - [ ] Troubleshooting guide
- [ ] Add inline code documentation (godoc comments)
- [ ] Create CONTRIBUTING.md for contributors
- [ ] Add LICENSE information (already created)
- [ ] Add example workflows and use cases

**Estimated Time**: 2-3 hours

--- 


## Dependencies

### Required Go Packages

```bash
# SQLite database (pure Go, no CGO required)
go get modernc.org/sqlite

# Alternative SQLite (requires CGO, faster but less portable)
# go get github.com/mattn/go-sqlite3

# Already installed:
# - github.com/spf13/cobra (CLI framework)
# - github.com/rotisserie/eris (error handling)
```

### Optional Enhancement Packages

```bash
# Table formatting
go get github.com/olekukonko/tablewriter

# Alternative: Modern TUI components
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/lipgloss

# Colored output
go get github.com/fatih/color
```

### External Dependencies (via flake.nix)

Already configured:
- `tmux` - Required for session management (users need to install)
- `git` - Required for workspace detection (users need to install)
- `fzf` or `peco` - Optional for interactive selection (users can install)

---

## Implementation Timeline

### Quick Reference

| Phase | Estimated Time | Priority |
|-------|---------------|----------|
| 1.1 - Project Structure | 30 min | High |
| 1.2 - Config Management | 2-3 hours | High |
| 1.3 - Database Schema | 45 min | High |
| 1.4 - Database Layer | 5-7 hours | High |
| 1.5 - Data Models | 45 min | High |
| 2.1 - Git Clone Management | 3-4 hours | High |
| 2.2 - Git Worktree Management | 3-4 hours | High |
| 2.3 - Git Branch Operations | 2-3 hours | High |
| 2.4 - Project Resolution | 2-3 hours | High |
| 2.5 - Fuzzy Finder Integration | 2-3 hours | Medium |
| 2.6.1 - Session Manager Interface | 1-2 hours | High |
| 2.6.2 - Tmux Backend Implementation | 3-4 hours | High |
| 2.7 - Workspace Folder Mgmt | 2-3 hours | High |
| 3.1 - Clone Command | 3-4 hours | High |
| 3.2 - Switch Command | 4-5 hours | High |
| 3.3 - List Command | 3-4 hours | High |
| 3.4 - Delete Command | 2-3 hours | High |
| 3.5 - Status Command | 2-3 hours | Medium |
| 3.6 - Fetch Command | 1-2 hours | Medium |
| 4.1 - Integration | 3-4 hours | High |
| 4.2 - Unit Testing | 6-8 hours | Medium |
| 4.3 - Integration Testing | 3-4 hours | Medium |
| 4.4 - Quality Checks | 2-3 hours | High |
| 5.1 - UX Enhancements | 3-4 hours | Low |
| 5.2 - Documentation | 3-4 hours | Medium |
| 5.3 - Build & Distribution | 2-3 hours | Low |

**Total Estimated Time**: 60-85 hours

### Suggested Implementation Order

**Phase 1: Foundation (10-13 hours)**
- Complete all infrastructure and database setup
- This provides the foundation for everything else

**Phase 2: Core Business Logic (17-24 hours)**
- Implement git operations (clone, worktree, branches)
- Implement project resolution and fuzzy finding
- Implement tmux session management
- Implement workspace folder management

**Phase 3: CLI Commands - MVP (12-16 hours)**
- Implement clone command
- Implement switch command (the most critical feature)
- Implement list command
- Implement delete command

**Phase 4: Testing & Integration (11-15 hours)**
- Wire everything together
- Add comprehensive tests
- Run quality checks

**Phase 5: Polish & Additional Features (8-11 hours)**
- Add status and fetch commands
- Enhance UX with colors and better formatting
- Write documentation
- Prepare for distribution

### MVP vs Full Feature Set

**Minimum Viable Product** (Phases 1-3, ~40-50 hours):
- Clone repositories into workspace
- Switch between branches with worktrees
- List all sessions
- Delete sessions/worktrees
- Basic fuzzy finding support

**Full Feature Set** (All Phases, ~60-85 hours):
- All MVP features
- Status command with git integration
- Fetch command for updating repos
- Enhanced UI with colors and formatting
- Comprehensive test coverage
- Full documentation
- Shell completions

---

## Success Criteria

### Minimum Viable Product (MVP)
- ✅ Can clone git repositories into workspace folder (`~/.sesh`)
- ✅ Can create git worktrees for different branches
- ✅ Can switch between branches with automatic worktree/session creation
- ✅ Fuzzy search for remote branches works
- ✅ Can list all projects, worktrees, and sessions
- ✅ Can delete worktrees and sessions
- ✅ Project resolution from CWD works correctly
- ✅ Sessions persist across restarts
- ✅ All core functionality has tests
- ✅ Code passes `task check`

### Future Enhancements
- Additional session manager backends (Zellij, Screen, Wezterm)
- Advanced fuzzy finding with branch previews
- Workspace tagging and search
- Automatic worktree cleanup for merged branches
- Shell integration (cd hook to auto-switch sessions)
- Remote session support (SSH)
- Branch creation with automatic remote push
- Pull request integration
- Session templates with custom commands
- Automatic dependency installation per worktree
- Plugin system for custom backends

---

## Risk Assessment

### Technical Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Git worktree conflicts | Medium | High | Proper error handling, clear messages |
| Bare repo corruption | Low | High | Regular git fsck, backup recommendations |
| Workspace disk space issues | Medium | Medium | Add cleanup commands, warn on low space |
| SQLite file corruption | Low | High | Regular backups, atomic writes |
| Tmux not installed | Medium | High | Clear error message, check on startup |
| Git auth failures (SSH/HTTPS) | Medium | High | Detect auth method, provide helpful errors |
| Platform-specific path issues | Low | Medium | Use Go's filepath package, test on multiple OS |
| Database migration failures | Low | High | Versioned migrations, backup before migrate |
| Fuzzy finder not available | Medium | Low | Implement fallback selection method |
| Branch name sanitization edge cases | Low | Low | Comprehensive test cases for special chars |

### Development Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Scope creep | High | High | Stick to MVP, defer enhancements |
| Git worktree complexity | Medium | High | Study git worktree thoroughly, test extensively |
| Underestimated complexity | High | Medium | Time estimates are generous, track actual time |
| Testing gaps | Medium | High | Write tests alongside code, not after |
| Cross-platform compatibility | Low | Medium | Test on Linux, macOS, and Windows |

---

## Next Steps

1. **Review this plan** - Adjust estimates and priorities as needed
2. **Set up project tracking** - Use GitHub issues, Jira, or similar
3. **Start with Phase 1.1** - Create project structure
4. **Implement incrementally** - Complete one phase before moving to next
5. **Test continuously** - Run `task check` frequently
6. **Commit regularly** - Small, focused commits with clear messages

---

## Questions to Consider

Before starting implementation:

1. **SQLite Driver**: Use `modernc.org/sqlite` (pure Go) or `mattn/go-sqlite3` (CGO, faster)?
   - **Recommendation**: `modernc.org/sqlite` for easier cross-compilation

2. **Worktree Naming**: Use branch name as-is or sanitize special characters?
   - **Recommendation**: Sanitize special characters (/, \, :, etc.) to filesystem-safe names

3. **Session Naming**: How to generate tmux session names from project + branch?
   - **Recommendation**: `{repo-name}:{branch}` (e.g., `sesh:main`, `myproject:feature-foo`)

4. **Default Branch**: How to determine default branch (main vs master)?
   - **Recommendation**: Query `git symbolic-ref refs/remotes/origin/HEAD`

5. **Worktree Location**: Where to create worktrees relative to bare repo?
   - **Recommendation**: Siblings of `.git/` directory (e.g., `~/.sesh/github.com/user/repo/main/`)

6. **Branch Cleanup**: Automatically delete worktrees for merged branches?
   - **Recommendation**: Not in MVP, add as future enhancement with `--merged` flag

7. **Fuzzy Finder Fallback**: What to do if fzf/peco not available?
   - **Recommendation**: Show numbered list, prompt for selection

8. **Remote vs Local Branches**: When fuzzy finding, show both or only remote?
   - **Recommendation**: Show all (local + remote), with clear indicators

9. **Git Auth**: How to handle SSH keys, HTTPS tokens, etc.?
   - **Recommendation**: Rely on git's credential helpers, don't store credentials

10. **Workspace Folder**: Allow multiple workspace folders or just one?
    - **Recommendation**: One workspace folder for MVP, configurable via `SESH_WORKSPACE`

11. **Concurrent Operations**: Handle multiple sesh instances modifying workspace?
    - **Recommendation**: SQLite handles locking, git handles repo locking, document limitations

12. **Session Manager Detection**: When to detect inside tmux (Switch vs Attach)?
    - **Recommendation**: Detect if already in tmux, use `switch-client` if inside, `attach-session` if outside

13. **Backend Auto-Detection**: What order to try backends when auto-detecting?
    - **Recommendation**: tmux (most popular) → zellij (modern) → screen (classic) → error

14. **Backend Registration**: Allow plugins to register new backends?
    - **Recommendation**: Not in MVP, but design interface to allow it in future

15. **No-Backend Mode**: Should sesh work without any session manager?
    - **Recommendation**: Yes, add "none" backend that just manages worktrees

---

## Design Decisions Summary

**Workspace Structure**:
```
~/.sesh/
  github.com/
    user/
      repo/
        .git/              # Bare repository
        main/              # Main branch worktree
        feature-foo/       # Feature branch worktree
```

**Session Backend Architecture**:
- Abstracted `SessionManager` interface
- Initial implementation: Tmux
- Future backends: Zellij, Screen, Wezterm, None
- Auto-detection with configuration override
- Each backend implements same interface for consistency

**Session Naming Convention**: `{repo-name}:{branch}`
- `sesh:main`
- `myproject:feature/foo` → `myproject:feature-foo` (sanitized)

**Command Behavior**:
- `sesh clone <url>` - Clone and create main worktree
- `sesh switch <branch>` - Switch to branch (CWD project)
- `sesh switch -p <project> <branch>` - Switch to branch (explicit project)
- `sesh switch -b <branch>` - Create new branch and switch
- `sesh switch` - Fuzzy find branches and switch

**Project Resolution Priority**:
1. Explicit project argument
2. Current working directory (find git repo)
3. Error if not found

---

**Document Version**: 2.1
**Last Updated**: 2025-11-23
**Status**: Ready for Review - Updated with worktree-based design and session backend abstraction
