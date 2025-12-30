-- Core project tracking
CREATE TABLE projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,          -- e.g., "github.com/user/repo"
    display_name TEXT NOT NULL,         -- e.g., "repo"
    remote_url TEXT NOT NULL,
    clone_path TEXT UNIQUE NOT NULL,    -- Actual path to cloned repo
    default_branch TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    last_fetched_at TEXT
);

-- Worktree tracking (includes primary worktree = the clone itself)
CREATE TABLE worktrees (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    branch TEXT NOT NULL,
    path TEXT UNIQUE NOT NULL,          -- Actual path to worktree
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,  -- TRUE for the original clone
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    last_accessed_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(project_id, branch)
);

-- Session tracking
CREATE TABLE sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    worktree_id INTEGER NOT NULL REFERENCES worktrees(id) ON DELETE CASCADE,
    session_name TEXT UNIQUE NOT NULL,
    backend TEXT NOT NULL,              -- 'tmux', 'code', 'cursor'
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    last_attached_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Session history stack for pop command
CREATE TABLE session_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    accessed_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Indexes
CREATE INDEX idx_worktrees_project ON worktrees(project_id);
CREATE INDEX idx_worktrees_path ON worktrees(path);
CREATE INDEX idx_sessions_worktree ON sessions(worktree_id);
CREATE INDEX idx_sessions_name ON sessions(session_name);
CREATE INDEX idx_history_time ON session_history(accessed_at DESC);

