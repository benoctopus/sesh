-- session_history table for tracking session access history (session stack)
-- This enables the "pop" command to switch back to previous sessions
CREATE TABLE IF NOT EXISTS session_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_name TEXT NOT NULL,          -- Name of the session (e.g., "repo-branch")
    project_name TEXT,                   -- Project name for reference
    branch TEXT,                         -- Branch name for reference
    accessed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_session_history_accessed_at ON session_history(accessed_at DESC);
CREATE INDEX idx_session_history_session_name ON session_history(session_name);
