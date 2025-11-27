package models

import "time"

// Project represents a git repository in the workspace
type Project struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`                   // e.g., "github.com/user/repo"
	RemoteURL   string     `json:"remote_url"`             // Git remote URL
	LocalPath   string     `json:"local_path"`             // Path to bare repo in workspace
	CreatedAt   time.Time  `json:"created_at"`             // When the project was cloned
	LastFetched *time.Time `json:"last_fetched,omitempty"` // Last time we fetched from remote
}

// Worktree represents a git worktree for a specific branch
type Worktree struct {
	ID        int       `json:"id"`
	ProjectID int       `json:"project_id"` // Foreign key to Project
	Branch    string    `json:"branch"`     // Branch/ref name
	Path      string    `json:"path"`       // Path to worktree directory
	IsMain    bool      `json:"is_main"`    // Is this the main worktree?
	CreatedAt time.Time `json:"created_at"` // When the worktree was created
	LastUsed  time.Time `json:"last_used"`  // Last time this worktree was accessed
}

// Session represents a tmux session tied to a worktree
type Session struct {
	ID              int       `json:"id"`
	WorktreeID      int       `json:"worktree_id"`       // Foreign key to Worktree
	TmuxSessionName string    `json:"tmux_session_name"` // Tmux session name
	CreatedAt       time.Time `json:"created_at"`        // When the session was created
	LastAttached    time.Time `json:"last_attached"`     // Last time we attached to this session
}

// SessionDetails is a composite type for queries that join sessions, worktrees, and projects
type SessionDetails struct {
	Session  *Session
	Worktree *Worktree
	Project  *Project
}

// SessionHistory represents an entry in the session access history (for the pop command)
type SessionHistory struct {
	ID          int       `json:"id"`
	SessionName string    `json:"session_name"` // Name of the session (e.g., "repo-branch")
	ProjectName string    `json:"project_name"` // Project name for reference
	Branch      string    `json:"branch"`       // Branch name for reference
	AccessedAt  time.Time `json:"accessed_at"`  // When the session was accessed
}
