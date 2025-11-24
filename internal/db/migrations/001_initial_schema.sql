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
